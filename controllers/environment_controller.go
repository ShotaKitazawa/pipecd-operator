package controllers

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	errors_ "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/services"
)

const (
	environmentControllerName = "envirionment"
	environmentFinalizerName  = "environment.finalizers.pipecd.kanatakita.com"
)

// EnvironmentReconciler reconciles a Environment object
type EnvironmentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipecdv1alpha1.Environment{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=environments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=environments/status,verbs=get;update;patch

func (r *EnvironmentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues(environmentControllerName, req.NamespacedName)

	/* Load Environment */
	var e pipecdv1alpha1.Environment
	if err := r.Get(ctx, req.NamespacedName, &e); err != nil {
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

	/* Generate Environment Service */
	environmentService, ctx, err := services.NewEnvironmentService(ctx, r.Client, req.Namespace, e.Spec.EncryptionKeyRef, e.Spec.ProjectId, e.Spec.Insecure)
	if err != nil {
		log.Info("cannot generate Environment WebService gRPC Client")
		return ctrl.Result{}, nil
	}

	// delete if DeletionTimestamp is zero && (Finalizers is none || Finalizers contain only environmentFinalizer)
	if !e.ObjectMeta.DeletionTimestamp.IsZero() &&
		(len(e.ObjectMeta.Finalizers) == 0 ||
			reflect.DeepEqual(e.ObjectMeta.Finalizers, []string{environmentFinalizerName})) {
		return r.reconcileDelete(ctx, log, &e, &cp, environmentService)
	}

	return r.reconcile(ctx, log, &e, &cp, environmentService)
}

func (r *EnvironmentReconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	e *pipecdv1alpha1.Environment,
	cp *pipecdv1alpha1.ControlPlane,
	environmentService *services.EnvironmentService,
) (ctrl.Result, error) {

	// create Environment
	environment, err := environmentService.CreateIfNotExists(ctx, e.Spec.Name, e.Spec.Description)
	if err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	// update Environment Object status
	if e.Status.EnvironmentId != environment.Id {
		e.Status.EnvironmentId = environment.Id
		if err := r.Status().Update(ctx, e); err != nil {
			log.Error(err, err.Error())
			return ctrl.Result{}, err
		}
		/* Record to event */
		r.Recorder.Eventf(e, corev1.EventTypeNormal, "Updated", "Update Environment.status.environmentId: %d", e.Status.EnvironmentId)
	}

	// add Environment finalizer to Environment Object
	patchEnv := generatePatchFinalizersObject(e.GroupVersionKind(), e.Namespace, e.Name, e.ObjectMeta.Finalizers)
	controllerutil.AddFinalizer(patchEnv, environmentFinalizerName)
	// TODO: 複数 controller が 1object を操作する場合 server-side apply だと conflict する
	if err := r.Patch(ctx, patchEnv, client.Apply, &client.PatchOptions{FieldManager: environmentControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	// add Environment finalizer to ControlPlane Object
	/* TODO
	patchCp := generatePatchFinalizersObject(cp.GroupVersionKind(), cp.Namespace, cp.Name, cp.ObjectMeta.Finalizers)
	controllerutil.AddFinalizer(cp, environmentFinalizerName)
	if err := r.Patch(ctx, patchCp, client.Merge, &client.PatchOptions{FieldManager: environmentControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}
	*/

	return ctrl.Result{}, nil
}

func (r *EnvironmentReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	e *pipecdv1alpha1.Environment,
	cp *pipecdv1alpha1.ControlPlane,
	environmentService *services.EnvironmentService,
) (ctrl.Result, error) {

	// delete Environment
	if err := environmentService.Delete(ctx, e.Status.EnvironmentId); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	// remove Environment finalizer from Environment Object
	controllerutil.RemoveFinalizer(e, environmentFinalizerName)
	patchEnv := generatePatchFinalizersObject(e.GroupVersionKind(), e.Namespace, e.Name, e.ObjectMeta.Finalizers)
	// controllerutil.RemoveFinalizer(patchEnv, environmentFinalizerName)
	// TODO: 複数 controller が 1object を操作する場合 server-side apply だと conflict する
	if err := r.Patch(ctx, patchEnv, client.Apply, &client.PatchOptions{FieldManager: environmentControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	// remove Environment finalizer from ControlPlane Object
	/* TODO
	patchCp := generatePatchFinalizersObject(cp.GroupVersionKind(), cp.Namespace, cp.Name, cp.ObjectMeta.Finalizers)
	if err := r.Patch(ctx, patchCp, client.Merge, &client.PatchOptions{FieldManager: environmentControllerName}); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}
	*/

	return ctrl.Result{}, nil
}
