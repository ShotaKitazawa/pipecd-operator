package controllers

import (
	"context"
	"sync"
	"time"

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

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/controlplane"
)

var reconcileTimeoutSecond = 30 * time.Second

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
	log := r.Log.WithValues("controlplane", req.NamespacedName)

	/* Load ControlPlane */
	var cp pipecdv1alpha1.ControlPlane
	log.Info("fetching ControlPlane Resource")
	if err := r.Get(ctx, req.NamespacedName, &cp); err != nil {
		if errors_.IsNotFound(err) {
			r.Log.Info("ControlPlane not found", "Namespace", req.Namespace, "Name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	/* TODO: use gorouting
		var wg sync.WaitGroup
		wg.Add(1)
		errStreamGeneral := make(chan error)
		go r.reconcileGeneral(ctx, &wg, errStreamGeneral, &cp)
		wg.Add(1)
		errStreamGateway := make(chan error)
		go r.reconcileGateway(ctx, &wg, errStreamGateway, &cp)
		wg.Add(1)
		errStreamApi := make(chan error)
		go r.reconcileApi(ctx, &wg, errStreamApi, &cp)

		terminated := make(chan interface{})
		go func() {
			wg.Wait()
			close(terminated)
		}()

		var err error
	L:
		for {
			select {
			case <-terminated: // ok
				break L
			case err = <-errStreamGeneral:
				<-terminated // wait until close all goroutine
				break L
			case err = <-errStreamGateway:
				<-terminated // wait until close all goroutine
				break L
			case err = <-errStreamApi:
				<-terminated // wait until close all goroutine
				break L
			case <-time.After(reconcileTimeoutSecond):
				err = fmt.Errorf("Reconcile process time out: something error occurred. (and occurring that leak goroutine)")
				break L
			default:
				time.Sleep(time.Microsecond)
			}
		}
		close(errStreamGeneral)
		close(errStreamGateway)
		close(errStreamApi)
		return ctrl.Result{}, err
	*/

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
		go r.reconcileGeneral(ctx, &wg, errStreamGeneral, &cp)
		select {
		case <-terminated:
		case err = <-errStreamGeneral:
			return ctrl.Result{}, err
		}
	}
	{ // gateway
		wg.Add(1)
		errStreamGateway := make(chan error)
		go r.reconcileGateway(ctx, &wg, errStreamGateway, &cp)
		select {
		case <-terminated:
		case err = <-errStreamGateway:
			return ctrl.Result{}, err
		}
	}
	{ // api
		wg.Add(1)
		errStreamApi := make(chan error)
		go r.reconcileApi(ctx, &wg, errStreamApi, &cp)
		select {
		case <-terminated:
		case err = <-errStreamApi:
			return ctrl.Result{}, err
		}
	}
	{ // web
		wg.Add(1)
		errStreamWeb := make(chan error)
		go r.reconcileWeb(ctx, &wg, errStreamWeb, &cp)
		select {
		case <-terminated:
		case err = <-errStreamWeb:
			return ctrl.Result{}, err
		}
	}
	{ // cache
		wg.Add(1)
		errStreamCache := make(chan error)
		go r.reconcileCache(ctx, &wg, errStreamCache, &cp)
		select {
		case <-terminated:
		case err = <-errStreamCache:
			return ctrl.Result{}, err
		}
	}
	{ // web
		wg.Add(1)
		errStreamOps := make(chan error)
		go r.reconcileOps(ctx, &wg, errStreamOps, &cp)
		select {
		case <-terminated:
		case err = <-errStreamOps:
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ControlPlaneReconciler) reconcileGeneral(ctx context.Context, wg *sync.WaitGroup, errStream chan<- error, cp *pipecdv1alpha1.ControlPlane) {
	defer (*wg).Done()
	log := r.Log.WithValues("component", "general")

	/* Generate ConfigMap (NamespacedName) */
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controlplane.ConfigMapName,
			Namespace: cp.Namespace,
		},
	}

	/* Apply ConfigMap */
	if _, err := ctrl.CreateOrUpdate(ctx, r, cm, func() (err error) {
		cm.BinaryData, err = controlplane.MakeConfigMapBinaryData(*cp)
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

	/* Generate Service (NamespacedName) */
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controlplane.ServiceName,
			Namespace: cp.Namespace,
		},
	}

	/* Apply Service */
	if _, err := ctrl.CreateOrUpdate(ctx, r, service, func() (err error) {
		service.Spec, err = controlplane.MakeServiceSpec(*cp)
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

func (r *ControlPlaneReconciler) reconcileGateway(ctx context.Context, wg *sync.WaitGroup, errStream chan<- error, cp *pipecdv1alpha1.ControlPlane) {
	defer (*wg).Done()
	log := r.Log.WithValues("component", "gateway")
	gatewayNN := controlplane.MakeGatewayNamespacedName(cp.Name, cp.Namespace)

	/* Generate gateway ConfigMap (NamespacedName) */
	gatewayConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayNN.Name,
			Namespace: gatewayNN.Namespace,
		},
	}

	/* Apply gateway ConfigMap */
	if _, err := ctrl.CreateOrUpdate(ctx, r, gatewayConfigMap, func() (err error) {
		gatewayConfigMap.BinaryData, err = controlplane.MakeGatewayConfigMapBinaryData(*cp)
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
		gatewayDeployment.Spec, err = controlplane.MakeGatewayDeploymentSpec(*cp, "inputHash_TODO")
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, gatewayDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure deployment is correct")
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
		gatewayService.Spec, err = controlplane.MakeGatewayServiceSpec(*cp)
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

func (r *ControlPlaneReconciler) reconcileApi(ctx context.Context, wg *sync.WaitGroup, errStream chan<- error, cp *pipecdv1alpha1.ControlPlane) {
	defer (*wg).Done()
	log := r.Log.WithValues("component", "api")
	apiNN := controlplane.MakeApiNamespacedName(cp.Name, cp.Namespace)

	/* Generate api Deployment (NamespacedName) */
	apiDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiNN.Name,
			Namespace: apiNN.Namespace,
		},
	}

	/* Apply api Deployment */
	if _, err := ctrl.CreateOrUpdate(ctx, r, apiDeployment, func() (err error) {
		apiDeployment.Spec, err = controlplane.MakeApiDeploymentSpec(*cp, "inputHash_TODO")
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, apiDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure deployment is correct")
		errStream <- err
		return
	}

	/* Get api Deployment from cluster */
	var apiDeploymentApplied appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: apiDeployment.Namespace, Name: apiDeployment.Name}, &apiDeploymentApplied); err != nil {
		log.Error(err, "unable to fetch Deployment")
		errStream <- client.IgnoreNotFound(err)
		return
	}

	/* Update status api Deployment */
	availableReplicas := apiDeploymentApplied.Status.AvailableReplicas
	if availableReplicas != cp.Status.AvailableApiReplicas {
		cp.Status.AvailableApiReplicas = availableReplicas
		if err := r.Status().Update(ctx, cp); err != nil {
			log.Error(err, "unable to update ControlPlane status")
			errStream <- err
			return
		}
		/* Record to event */
		r.Recorder.Eventf(cp, corev1.EventTypeNormal, "Updated", "Update controlPlane.Status.AvailableApiReplicas: %d", cp.Status.AvailableApiReplicas)
	}

	/* Generate api Service (NamespacedName) */
	apiService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiNN.Name,
			Namespace: apiNN.Namespace,
		},
	}

	/* Apply api Service */
	if _, err := ctrl.CreateOrUpdate(ctx, r, apiService, func() (err error) {
		apiService.Spec, err = controlplane.MakeApiServiceSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, apiService, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Service")
			return err
		}
		/* Get gateway Service from cluster */
		var apiServiceApplied v1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: apiService.Namespace, Name: apiService.Name}, &apiServiceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		apiService.Spec.ClusterIP = apiServiceApplied.Spec.ClusterIP

		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		errStream <- err
		return
	}
}

func (r *ControlPlaneReconciler) reconcileWeb(ctx context.Context, wg *sync.WaitGroup, errStream chan<- error, cp *pipecdv1alpha1.ControlPlane) {
	defer (*wg).Done()
	log := r.Log.WithValues("component", "web")
	webNN := controlplane.MakeWebNamespacedName(cp.Name, cp.Namespace)

	/* Generate web Deployment (NamespacedName) */
	webDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webNN.Name,
			Namespace: webNN.Namespace,
		},
	}

	/* Apply web Deployment */
	if _, err := ctrl.CreateOrUpdate(ctx, r, webDeployment, func() (err error) {
		webDeployment.Spec, err = controlplane.MakeWebDeploymentSpec(*cp, "inputHash_TODO")
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, webDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure deployment is correct")
		errStream <- err
		return
	}

	/* Get web Deployment from cluster */
	var webDeploymentApplied appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: webDeployment.Namespace, Name: webDeployment.Name}, &webDeploymentApplied); err != nil {
		log.Error(err, "unable to fetch Deployment")
		errStream <- client.IgnoreNotFound(err)
		return
	}

	/* Update status web Deployment */
	availableReplicas := webDeploymentApplied.Status.AvailableReplicas
	if availableReplicas != cp.Status.AvailableWebReplicas {
		cp.Status.AvailableWebReplicas = availableReplicas
		if err := r.Status().Update(ctx, cp); err != nil {
			log.Error(err, "unable to update ControlPlane status")
			errStream <- err
			return
		}
		/* Record to event */
		r.Recorder.Eventf(cp, corev1.EventTypeNormal, "Updated", "Update controlPlane.Status.AvailableWebReplicas: %d", cp.Status.AvailableWebReplicas)
	}

	/* Generate web Service (NamespacedName) */
	webService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webNN.Name,
			Namespace: webNN.Namespace,
		},
	}

	/* Apply web Service */
	if _, err := ctrl.CreateOrUpdate(ctx, r, webService, func() (err error) {
		webService.Spec, err = controlplane.MakeWebServiceSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, webService, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Service")
			return err
		}
		/* Get gateway Service from cluster */
		var webServiceApplied v1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: webService.Namespace, Name: webService.Name}, &webServiceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		webService.Spec.ClusterIP = webServiceApplied.Spec.ClusterIP

		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		errStream <- err
		return
	}
}

func (r *ControlPlaneReconciler) reconcileCache(ctx context.Context, wg *sync.WaitGroup, errStream chan<- error, cp *pipecdv1alpha1.ControlPlane) {
	defer (*wg).Done()
	log := r.Log.WithValues("component", "cache")
	cacheNN := controlplane.MakeCacheNamespacedName(cp.Name, cp.Namespace)

	/* Generate cache Deployment (NamespacedName) */
	cacheDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cacheNN.Name,
			Namespace: cacheNN.Namespace,
		},
	}

	/* Apply cache Deployment */
	if _, err := ctrl.CreateOrUpdate(ctx, r, cacheDeployment, func() (err error) {
		cacheDeployment.Spec, err = controlplane.MakeCacheDeploymentSpec(*cp)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, cacheDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure deployment is correct")
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
		cacheService.Spec, err = controlplane.MakeCacheServiceSpec(*cp)
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

func (r *ControlPlaneReconciler) reconcileOps(ctx context.Context, wg *sync.WaitGroup, errStream chan<- error, cp *pipecdv1alpha1.ControlPlane) {
	defer (*wg).Done()
	log := r.Log.WithValues("component", "ops")
	opsNN := controlplane.MakeOpsNamespacedName(cp.Name, cp.Namespace)

	/* Generate ops Deployment (NamespacedName) */
	opsDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opsNN.Name,
			Namespace: opsNN.Namespace,
		},
	}

	/* Apply ops Deployment */
	if _, err := ctrl.CreateOrUpdate(ctx, r, opsDeployment, func() (err error) {
		opsDeployment.Spec, err = controlplane.MakeOpsDeploymentSpec(*cp, "inputHash_TODO")
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(cp, opsDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from ControlPlane to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure deployment is correct")
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
		opsService.Spec, err = controlplane.MakeOpsServiceSpec(*cp)
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
