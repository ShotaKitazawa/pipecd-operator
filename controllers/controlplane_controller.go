package controllers

import (
	"context"
	"reflect"
	"sync"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	errors_ "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/object"
)

const (
	controlPlaneControllerName = "controlplane"
	controlPlaneFinalizerName  = "controlplane.finalizers.pipecd.kanatakita.com"
)

// ControlPlaneReconciler reconciles a ControlPlane object
type ControlPlaneReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *ControlPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipecdv1alpha1.ControlPlane{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=controlplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=controlplanes/status,verbs=get;update;patch

func (r *ControlPlaneReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues(controlPlaneControllerName, req.NamespacedName)

	/* Load ControlPlane */
	var cp pipecdv1alpha1.ControlPlane
	log.Info("fetching ControlPlane Resource")
	if err := r.Get(ctx, req.NamespacedName, &cp); err != nil {
		if errors_.IsNotFound(err) {
			log.Info("ControlPlane not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// delete if DeletionTimestamp is zero && (Finalizers is none || Finalizers contain only controlPlaneFinalizer)
	if !cp.ObjectMeta.DeletionTimestamp.IsZero() &&
		(len(cp.ObjectMeta.Finalizers) == 0 ||
			reflect.DeepEqual(cp.ObjectMeta.Finalizers, []string{controlPlaneFinalizerName})) {
		return r.reconcileDelete(ctx, log, &cp)
	}

	return r.reconcile(ctx, log, &cp)
}

func (r *ControlPlaneReconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	cp *pipecdv1alpha1.ControlPlane,
) (ctrl.Result, error) {

	// TODO: use goroutine

	var err error
	var wg sync.WaitGroup
	terminated := make(chan interface{})
	go func() {
		wg.Wait()
		close(terminated)
	}()
	{ // general
		wg.Add(1)
		errStreamGeneral := make(chan error)
		go r.reconcileGeneral(ctx, log, &wg, errStreamGeneral, cp)
		select {
		case <-terminated:
		case err = <-errStreamGeneral:
			return ctrl.Result{}, err
		}
	}
	{ // gateway
		wg.Add(1)
		errStreamGateway := make(chan error)
		go r.reconcileGateway(ctx, log, &wg, errStreamGateway, cp)
		select {
		case <-terminated:
		case err = <-errStreamGateway:
			return ctrl.Result{}, err
		}
	}
	{ // server
		wg.Add(1)
		errStreamApi := make(chan error)
		go r.reconcileServer(ctx, log, &wg, errStreamApi, cp)
		select {
		case <-terminated:
		case err = <-errStreamApi:
			return ctrl.Result{}, err
		}
	}
	{ // cache
		wg.Add(1)
		errStreamCache := make(chan error)
		go r.reconcileCache(ctx, log, &wg, errStreamCache, cp)
		select {
		case <-terminated:
		case err = <-errStreamCache:
			return ctrl.Result{}, err
		}
	}
	{ // web
		wg.Add(1)
		errStreamOps := make(chan error)
		go r.reconcileOps(ctx, log, &wg, errStreamOps, cp)
		select {
		case <-terminated:
		case err = <-errStreamOps:
			return ctrl.Result{}, err
		}
	}

	// add ControlPlane finalizer to ControlPlane Object
	controllerutil.AddFinalizer(cp, controlPlaneFinalizerName)
	patch := generatePatchFinalizersObject(cp.GroupVersionKind(), cp.Namespace, cp.Name, cp.ObjectMeta.Finalizers)
	if err := r.Patch(ctx, patch, client.Merge, &client.PatchOptions{FieldManager: controlPlaneControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ControlPlaneReconciler) reconcileGeneral(
	ctx context.Context,
	log logr.Logger,
	wg *sync.WaitGroup,
	errStream chan<- error,
	cp *pipecdv1alpha1.ControlPlane,
) {
	defer (*wg).Done()
	log = log.WithValues("component", "general")

	/* Apply ConfigMap */
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      object.ConfigMapName,
			Namespace: cp.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, cm, func() (err error) {
		cm.BinaryData, err = object.MakeConfigMapBinaryData(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, cm, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to ConfigMap")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure ConfigMap is correct")
		errStream <- err
		return
	}

	/* Apply server Secret */
	serverSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      object.SecretName,
			Namespace: cp.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, serverSecret, func() (err error) {
		serverSecret.StringData = object.MakeSecretData(*cp)
		if err := ctrl.SetControllerReference(cp, serverSecret, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Secret")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Secret is correct")
		errStream <- err
		return
	}

	/* Apply Service */
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      object.ServiceName,
			Namespace: cp.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, service, func() (err error) {
		service.Spec, err = object.MakeServiceSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, service, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Service")
			return err
		}
		/* Get gateway Service from cluster */
		var serviceApplied v1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: service.Namespace, Name: service.Name}, &serviceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		service.Spec.ClusterIP = serviceApplied.Spec.ClusterIP

		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		errStream <- err
		return
	}
}

func (r *ControlPlaneReconciler) reconcileGateway(
	ctx context.Context,
	log logr.Logger,
	wg *sync.WaitGroup,
	errStream chan<- error,
	cp *pipecdv1alpha1.ControlPlane,
) {
	defer (*wg).Done()
	log = log.WithValues("component", "gateway")
	gatewayNN := object.MakeGatewayNamespacedName(cp.Name, cp.Namespace)

	/* Generate gateway ConfigMap (NamespacedName) */
	gatewayConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayNN.Name,
			Namespace: gatewayNN.Namespace,
		},
	}

	/* Apply gateway ConfigMap */
	if _, err := ctrl.CreateOrUpdate(ctx, r, gatewayConfigMap, func() (err error) {
		gatewayConfigMap.Data, err = object.MakeGatewayConfigMapData(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, gatewayConfigMap, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to ConfigMap")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure ConfigMap is correct")
		errStream <- err
		return
	}

	/* Generate gateway Deployment (NamespacedName) */
	gatewayDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayNN.Name,
			Namespace: gatewayNN.Namespace,
		},
	}

	/* Apply gateway Deployment */
	if _, err := ctrl.CreateOrUpdate(ctx, r, gatewayDeployment, func() (err error) {
		gatewayDeployment.Spec, err = object.MakeGatewayDeploymentSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, gatewayDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Deployment is correct")
		errStream <- err
		return
	}

	/* Get gateway Deployment from cluster */
	var gatewayDeploymentApplied appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: gatewayDeployment.Namespace, Name: gatewayDeployment.Name}, &gatewayDeploymentApplied); err != nil {
		log.Error(err, "unable to fetch Deployment")
		errStream <- client.IgnoreNotFound(err)
		return
	}

	/* Update status gateway Deployment */
	availableReplicas := gatewayDeploymentApplied.Status.AvailableReplicas
	if availableReplicas != cp.Status.AvailableGatewayReplicas {
		cp.Status.AvailableGatewayReplicas = availableReplicas
		if err := r.Status().Update(ctx, cp); err != nil {
			log.Error(err, "unable to update ControlPlane status")
			errStream <- err
			return
		}
		/* Record to event */
		r.Recorder.Eventf(cp, corev1.EventTypeNormal, "Updated", "Update controlPlane.Status.AvailableGatewayReplicas: %d", cp.Status.AvailableGatewayReplicas)
	}

	/* Generate gateway Service (NamespacedName) */
	gatewayService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayNN.Name,
			Namespace: gatewayNN.Namespace,
		},
	}

	/* Apply gateway Service */
	if _, err := ctrl.CreateOrUpdate(ctx, r, gatewayService, func() (err error) {
		gatewayService.Spec, err = object.MakeGatewayServiceSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, gatewayService, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Service")
			return err
		}

		/* Get gateway Service from cluster */
		var gatewayServiceApplied v1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: gatewayService.Namespace, Name: gatewayService.Name}, &gatewayServiceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		gatewayService.Spec.ClusterIP = gatewayServiceApplied.Spec.ClusterIP

		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		errStream <- err
		return
	}
}

func (r *ControlPlaneReconciler) reconcileServer(
	ctx context.Context,
	log logr.Logger,
	wg *sync.WaitGroup,
	errStream chan<- error,
	cp *pipecdv1alpha1.ControlPlane,
) {
	defer (*wg).Done()
	log = log.WithValues("component", "server")
	serverNN := object.MakeServerNamespacedName(cp.Name, cp.Namespace)

	/* Apply server Deployment */
	serverDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverNN.Name,
			Namespace: serverNN.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, serverDeployment, func() (err error) {
		serverDeployment.Spec, err = object.MakeServerDeploymentSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, serverDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Deployment is correct")
		errStream <- err
		return
	}

	/* Get server Deployment from cluster */
	var serverDeploymentApplied appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: serverDeployment.Namespace, Name: serverDeployment.Name}, &serverDeploymentApplied); err != nil {
		log.Error(err, "unable to fetch Deployment")
		errStream <- client.IgnoreNotFound(err)
		return
	}

	/* Update status server Deployment */
	availableReplicas := serverDeploymentApplied.Status.AvailableReplicas
	if availableReplicas != cp.Status.AvailableServerReplicas {
		cp.Status.AvailableServerReplicas = availableReplicas
		if err := r.Status().Update(ctx, cp); err != nil {
			log.Error(err, "unable to update ControlPlane status")
			errStream <- err
			return
		}
		// record to event
		r.Recorder.Eventf(cp, corev1.EventTypeNormal, "Updated", "Update controlPlane.Status.AvailableServerReplicas: %d", cp.Status.AvailableServerReplicas)
	}

	/* Apply server Service */
	serverService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serverNN.Name,
			Namespace: serverNN.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, serverService, func() (err error) {
		serverService.Spec, err = object.MakeServerServiceSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, serverService, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Service")
			return err
		}
		// get gateway Service from cluster
		var serverServiceApplied v1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: serverService.Namespace, Name: serverService.Name}, &serverServiceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		serverService.Spec.ClusterIP = serverServiceApplied.Spec.ClusterIP

		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		errStream <- err
		return
	}
}

func (r *ControlPlaneReconciler) reconcileCache(
	ctx context.Context,
	log logr.Logger,
	wg *sync.WaitGroup,
	errStream chan<- error,
	cp *pipecdv1alpha1.ControlPlane,
) {
	defer (*wg).Done()
	log = log.WithValues("component", "cache")
	cacheNN := object.MakeCacheNamespacedName(cp.Name, cp.Namespace)

	/* Generate cache Deployment (NamespacedName) */
	cacheDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cacheNN.Name,
			Namespace: cacheNN.Namespace,
		},
	}

	/* Apply cache Deployment */
	if _, err := ctrl.CreateOrUpdate(ctx, r, cacheDeployment, func() (err error) {
		cacheDeployment.Spec, err = object.MakeCacheDeploymentSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, cacheDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Deployment is correct")
		errStream <- err
		return
	}

	/* Get cache Deployment from cluster */
	var cacheDeploymentApplied appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: cacheDeployment.Namespace, Name: cacheDeployment.Name}, &cacheDeploymentApplied); err != nil {
		log.Error(err, "unable to fetch Deployment")
		errStream <- client.IgnoreNotFound(err)
		return
	}

	/* Update status cache Deployment */
	availableReplicas := cacheDeploymentApplied.Status.AvailableReplicas
	if availableReplicas != cp.Status.AvailableCacheReplicas {
		cp.Status.AvailableCacheReplicas = availableReplicas
		if err := r.Status().Update(ctx, cp); err != nil {
			log.Error(err, "unable to update ControlPlane status")
			errStream <- err
			return
		}
		/* Record to event */
		r.Recorder.Eventf(cp, corev1.EventTypeNormal, "Updated", "Update controlPlane.Status.AvailableCacheReplicas: %d", cp.Status.AvailableCacheReplicas)
	}

	/* Generate cache Service (NamespacedName) */
	cacheService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cacheNN.Name,
			Namespace: cacheNN.Namespace,
		},
	}

	/* Apply cache Service */
	if _, err := ctrl.CreateOrUpdate(ctx, r, cacheService, func() (err error) {
		cacheService.Spec, err = object.MakeCacheServiceSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, cacheService, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Service")
			return err
		}
		/* Get gateway Service from cluster */
		var cacheServiceApplied v1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: cacheService.Namespace, Name: cacheService.Name}, &cacheServiceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		cacheService.Spec.ClusterIP = cacheServiceApplied.Spec.ClusterIP

		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		errStream <- err
		return
	}
}

func (r *ControlPlaneReconciler) reconcileOps(
	ctx context.Context,
	log logr.Logger,
	wg *sync.WaitGroup,
	errStream chan<- error,
	cp *pipecdv1alpha1.ControlPlane,
) {
	defer (*wg).Done()
	log = log.WithValues("component", "ops")
	opsNN := object.MakeOpsNamespacedName(cp.Name, cp.Namespace)

	/* Generate ops Deployment (NamespacedName) */
	opsDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opsNN.Name,
			Namespace: opsNN.Namespace,
		},
	}

	/* Apply ops Deployment */
	if _, err := ctrl.CreateOrUpdate(ctx, r, opsDeployment, func() (err error) {
		opsDeployment.Spec, err = object.MakeOpsDeploymentSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, opsDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Deployment is correct")
		errStream <- err
		return
	}

	/* Get ops Deployment from cluster */
	var opsDeploymentApplied appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: opsDeployment.Namespace, Name: opsDeployment.Name}, &opsDeploymentApplied); err != nil {
		log.Error(err, "unable to fetch Deployment")
		errStream <- client.IgnoreNotFound(err)
		return
	}

	/* Update status ops Deployment */
	availableReplicas := opsDeploymentApplied.Status.AvailableReplicas
	if availableReplicas != cp.Status.AvailableOpsReplicas {
		cp.Status.AvailableOpsReplicas = availableReplicas
		if err := r.Status().Update(ctx, cp); err != nil {
			log.Error(err, "unable to update ControlPlane status")
			errStream <- err
			return
		}
		/* Record to event */
		r.Recorder.Eventf(cp, corev1.EventTypeNormal, "Updated", "Update controlPlane.Status.AvailableOpsReplicas: %d", cp.Status.AvailableOpsReplicas)
	}

	/* Generate ops Service (NamespacedName) */
	opsService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opsNN.Name,
			Namespace: opsNN.Namespace,
		},
	}

	/* Apply ops Service */
	if _, err := ctrl.CreateOrUpdate(ctx, r, opsService, func() (err error) {
		opsService.Spec, err = object.MakeOpsServiceSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, opsService, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Service")
			return err
		}
		/* Get gateway Service from cluster */
		var opsServiceApplied v1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: opsService.Namespace, Name: opsService.Name}, &opsServiceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		opsService.Spec.ClusterIP = opsServiceApplied.Spec.ClusterIP

		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		errStream <- err
		return
	}
}

func (r *ControlPlaneReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	cp *pipecdv1alpha1.ControlPlane,
) (ctrl.Result, error) {

	// remove ControlPlane finalizer from ControlPlane Object
	controllerutil.RemoveFinalizer(cp, controlPlaneFinalizerName)
	patch := generatePatchFinalizersObject(cp.GroupVersionKind(), cp.Namespace, cp.Name, cp.ObjectMeta.Finalizers)
	// TODO: 複数 controller が 1object を操作する場合 server-side apply だと conflict する
	if err := r.Patch(ctx, patch, client.Apply, &client.PatchOptions{FieldManager: controlPlaneControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
