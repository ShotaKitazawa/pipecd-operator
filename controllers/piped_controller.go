package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	errors_ "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/object"
	"github.com/ShotaKitazawa/pipecd-operator/services"
)

const (
	pipedControllerName = "piped"
	pipedFinalizerName  = "piped.finalizers.pipecd.kanatakita.com"
)

var (
	pipedKeyCache string
)

// PipedReconciler reconciles a Piped object
type PipedReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *PipedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipecdv1alpha1.Piped{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=pipeds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=pipeds/status,verbs=get;update;patch

func (r *PipedReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues(pipedControllerName, req.NamespacedName)

	/* Load Piped Object */
	var p pipecdv1alpha1.Piped
	if err := r.Get(ctx, req.NamespacedName, &p); err != nil {
		if errors_.IsNotFound(err) {
			log.Info("Piped not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	/* Load Environment Object */
	var e pipecdv1alpha1.Environment
	if err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: p.Spec.EnvironmentRef.ObjectName}, &e); err != nil {
		if errors_.IsNotFound(err) {
			log.Info("Environment not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	/* Load ControlPlane */
	var cpList pipecdv1alpha1.ControlPlaneList
	if err := r.List(ctx, &cpList); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	} else if len(cpList.Items) == 0 {
		log.Info("ControlPlane not found")
		return ctrl.Result{}, nil
	}
	cp := cpList.Items[0]

	/* Generate Piped Service */
	pipedService, ctx, err := services.NewPipedService(ctx, r.Client, req.Namespace, p.Spec.EncryptionKeyRef, p.Spec.ProjectId, p.Spec.Insecure)
	if err != nil {
		log.Info("cannot generate Piped WebService gRPC Client")
		return ctrl.Result{}, nil
	}

	/* Generate Environment Service */
	environmentService, ctx, err := services.NewEnvironmentService(ctx, r.Client, req.Namespace, p.Spec.EncryptionKeyRef, p.Spec.ProjectId, p.Spec.Insecure)
	if err != nil {
		log.Info("cannot generate Environment WebService gRPC Client")
		return ctrl.Result{}, nil
	}

	// delete if DeletionTimestamp is zero && (Finalizers is none || Finalizers contain only pipedFinalizer)
	if !p.ObjectMeta.DeletionTimestamp.IsZero() &&
		(len(p.ObjectMeta.Finalizers) == 0 ||
			reflect.DeepEqual(p.ObjectMeta.Finalizers, []string{pipedFinalizerName})) {

		return r.reconcileDelete(ctx, log, &p, &e, &cp, environmentService, pipedService)
	}

	return r.reconcile(ctx, log, &p, &e, &cp, environmentService, pipedService)
}

func (r *PipedReconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	p *pipecdv1alpha1.Piped,
	e *pipecdv1alpha1.Environment,
	cp *pipecdv1alpha1.ControlPlane,
	environmentService *services.EnvironmentService,
	pipedService *services.PipedService,
) (ctrl.Result, error) {

	// list PipeCD Piped & create if Piped does not exist
	piped, err := pipedService.FindByName(ctx, p.Spec.Name)
	if err != nil {
		environment, err := environmentService.Find(ctx, p.Spec.EnvironmentRef.Name)
		if err != nil {
			log.Error(err, err.Error())
			return ctrl.Result{}, err
		}
		p, err := pipedService.Create(ctx, p.Spec.Name, p.Spec.Description, environment.Id)
		if err != nil {
			log.Error(err, err.Error())
			return ctrl.Result{}, err
		}
		piped = p
		pipedKeyCache = piped.Key
	} else {
		if pipedKeyCache == "" {
			err := fmt.Errorf("pipedKeyCache is none. Please disable piped configuration from WebUI.")
			log.Error(err, err.Error())
			return ctrl.Result{}, err
		}
		piped.Key = pipedKeyCache
	}

	// generate K8s Objects for Piped
	if result, err := r.reconcilePipedObject(ctx, log, p, piped.Id, pipedKeyCache, piped.EnvIds); err != nil {
		log.Error(err, err.Error())
		return result, err
	}

	// add Piped finalizer to Piped Object
	controllerutil.AddFinalizer(p, pipedFinalizerName)
	patchPiped := generatePatchFinalizersObject(p.GroupVersionKind(), p.Namespace, p.Name, p.ObjectMeta.Finalizers)
	// TODO: 複数 controller が 1object を操作する場合 server-side apply だと conflict する
	if err := r.Patch(ctx, patchPiped, client.Apply, &client.PatchOptions{FieldManager: pipedControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	// add Piped finalizer to Environment Object
	/* TODO
	patchEnv := generatePatchFinalizersObject(e.GroupVersionKind(), e.Namespace, e.Name, e.ObjectMeta.Finalizers)
	controllerutil.AddFinalizer(patchEnv, pipedFinalizerName)
	if err := r.Patch(ctx, patchEnv, client.Merge, &client.PatchOptions{FieldManager: pipedControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}
	*/

	// add Piped finalizer to ControlPlane Object
	/* TODO
	patchCp := generatePatchFinalizersObject(cp.GroupVersionKind(), cp.Namespace, cp.Name, cp.ObjectMeta.Finalizers)
	controllerutil.AddFinalizer(patchCp, pipedFinalizerName)
	if err := r.Patch(ctx, patchCp, client.Merge, &client.PatchOptions{FieldManager: pipedControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}
	*/

	return ctrl.Result{}, nil
}

func (r *PipedReconciler) reconcilePipedObject(
	ctx context.Context,
	log logr.Logger,
	p *pipecdv1alpha1.Piped,
	pipedId,
	pipedKey string,
	environmentIds []string,
) (ctrl.Result, error) {
	log = log.WithValues("component", "piped")
	pipedNN := object.MakePipedNamespacedName(p.Name, p.Namespace)

	/* Generate & Apply piped ServiceAccount */
	pipedServiceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipedNN.Name,
			Namespace: pipedNN.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, pipedServiceAccount, func() (err error) {
		if err := ctrl.SetControllerReference(p, pipedServiceAccount, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Piped to ServiceAccount")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure ServiceAccount is correct")
		return ctrl.Result{}, err
	}

	/* Generate & Apply piped Secret */
	pipedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipedNN.Name,
			Namespace: pipedNN.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, pipedSecret, func() (err error) {
		pipedSecret.StringData = object.MakePipedSecretData(*p, pipedKey)
		if err := ctrl.SetControllerReference(p, pipedSecret, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Piped to Secret")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Secret is correct")
		return ctrl.Result{}, err
	}

	/* Generate & Apply piped ConfigMap */
	pipedConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipedNN.Name,
			Namespace: pipedNN.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, pipedConfigMap, func() (err error) {
		pipedConfigMap.Data, err = object.MakePipedConfigMapData(*p, pipedId)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(p, pipedConfigMap, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Piped to ConfigMap")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure ConfigMap is correct")
		return ctrl.Result{}, err
	}

	/* Generate & Apply piped ClusterRole */
	// Memo: cannot set ownerReferences, because ClusterRole is cluster-wide Resource.
	pipedClusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: pipedNN.Name,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, pipedClusterRole, func() (err error) {
		pipedClusterRole.Rules = object.MakePipedClusterRoleRules(*p)
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure ClusterRole is correct")
		return ctrl.Result{}, err
	}

	/* Generate & Apply piped ClusterRoleBinding */
	// Memo: cannot set ownerReferences, because ClusterRoleBinding is cluster-wide Resource.
	pipedClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: pipedNN.Name,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, pipedClusterRoleBinding, func() (err error) {
		pipedClusterRoleBinding.RoleRef = object.MakePipedClusterRoleBindingRoleRef(*p)
		pipedClusterRoleBinding.Subjects = object.MakePipedClusterRoleBindingSubjects(*p)
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure ClusterRoleBinding is correct")
		return ctrl.Result{}, err
	}

	/* Generate & Apply piped Service */
	pipedService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipedNN.Name,
			Namespace: pipedNN.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, pipedService, func() (err error) {
		pipedService.Spec, err = object.MakePipedServiceSpec(*p)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(p, pipedService, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Piped to Service")
			return err
		}
		// Get piped Service from cluster
		var pipedServiceApplied corev1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: pipedService.Namespace, Name: pipedService.Name}, &pipedServiceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		pipedService.Spec.ClusterIP = pipedServiceApplied.Spec.ClusterIP
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		return ctrl.Result{}, err
	}

	/* Generate & Apply piped Deployment */
	pipedDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipedNN.Name,
			Namespace: pipedNN.Namespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, r, pipedDeployment, func() (err error) {
		pipedDeployment.Spec, err = object.MakePipedDeploymentSpec(*p, pipedId)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(p, pipedDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Piped to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Deployment is correct")
		return ctrl.Result{}, err
	}

	/* Get piped Deployment from cluster */
	var pipedDeploymentApplied appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: pipedDeployment.Namespace, Name: pipedDeployment.Name}, &pipedDeploymentApplied); err != nil {
		log.Error(err, "unable to fetch Deployment")
		return ctrl.Result{}, err
	}

	/* Update status piped Deployment */
	availableReplicas := pipedDeploymentApplied.Status.AvailableReplicas
	if p.Status.AvailableReplicas != availableReplicas {
		p.Status.AvailableReplicas = availableReplicas
		if err := r.Status().Update(ctx, p); err != nil {
			log.Error(err, "unable to update Piped status")
			return ctrl.Result{}, err
		}
		/* Record to event */
		r.Recorder.Eventf(p, corev1.EventTypeNormal, "Updated", "Update Piped.status.availablePipedReplicas: %d", p.Status.AvailableReplicas)
	}
	if p.Status.PipedId != pipedId {
		p.Status.PipedId = pipedId
		if err := r.Status().Update(ctx, p); err != nil {
			log.Error(err, "unable to update Piped status")
			return ctrl.Result{}, err
		}
		/* Record to event */
		r.Recorder.Eventf(p, corev1.EventTypeNormal, "Updated", "Update Piped.status.pipedId: %d", p.Status.AvailableReplicas)
	}
	envIdsSlice := strings.Join(environmentIds, ",")
	if p.Status.EnvironmentIds != envIdsSlice {
		p.Status.EnvironmentIds = envIdsSlice
		if err := r.Status().Update(ctx, p); err != nil {
			log.Error(err, "unable to update Piped status")
			return ctrl.Result{}, err
		}
		/* Record to event */
		r.Recorder.Eventf(p, corev1.EventTypeNormal, "Updated", "Update Piped.status.envIds: %d", p.Status.EnvironmentIds)
	}

	return ctrl.Result{}, nil
}

func (r *PipedReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	p *pipecdv1alpha1.Piped,
	e *pipecdv1alpha1.Environment,
	cp *pipecdv1alpha1.ControlPlane,
	environmentService *services.EnvironmentService,
	pipedService *services.PipedService,
) (ctrl.Result, error) {

	// disable Piped
	if p.Status.PipedId != "" {
		if err := pipedService.Delete(ctx, p.Status.PipedId); err != nil {
			log.Error(err, err.Error())
			return ctrl.Result{}, err
		}
	}

	// remove Piped finalizer from Piped Object
	controllerutil.RemoveFinalizer(p, pipedFinalizerName)
	patchPiped := generatePatchFinalizersObject(p.GroupVersionKind(), p.Namespace, p.Name, p.ObjectMeta.Finalizers)
	// TODO: 複数 controller が 1object を操作する場合 server-side apply だと conflict する
	if err := r.Patch(ctx, patchPiped, client.Apply, &client.PatchOptions{FieldManager: pipedControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	// remove Piped finalizer from Environment Object
	/* TODO
	patchEnv := generatePatchFinalizersObject(e.GroupVersionKind(), e.Namespace, e.Name, e.ObjectMeta.Finalizers)
	if err := r.Patch(ctx, patchEnv, client.Merge, &client.PatchOptions{FieldManager: pipedControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}
	*/

	// remove Piped finalizer from ControlPlane Object
	/* TODO
	patchCp := generatePatchFinalizersObject(cp.GroupVersionKind(), cp.Namespace, cp.Name, cp.ObjectMeta.Finalizers)
	if err := r.Patch(ctx, patchCp, client.Merge, &client.PatchOptions{FieldManager: pipedControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}
	*/

	return ctrl.Result{}, nil
}
