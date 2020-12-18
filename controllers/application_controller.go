package controllers

import (
	"context"

	"github.com/go-logr/logr"
	errors_ "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/services"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipecdv1alpha1.Application{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=applications/status,verbs=get;update;patch

func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("application", req.NamespacedName)

	/* Load Application */
	var a pipecdv1alpha1.Application
	log.Info("fetching Application Resource")
	if err := r.Get(ctx, req.NamespacedName, &a); err != nil {
		if errors_.IsNotFound(err) {
			log.Info("Application not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	/* Generate Application Service */
	applicationService, ctx, err := services.NewApplicationService(ctx, r.Client, req.Namespace, a.Spec.ProjectId, a.Spec.Insecure)
	if err != nil {
		log.Info("cannot generate Application WebService gRPC Client")
		return ctrl.Result{}, nil
	}

	if !a.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, &a, applicationService)
	}

	return r.reconcile(ctx, log, &a, applicationService)
}

func (r *ApplicationReconciler) reconcile(
	ctx context.Context,
	log logr.Logger,
	a *pipecdv1alpha1.Application,
	applicationService *services.ApplicationService,
) (ctrl.Result, error) {
	// create Application
	// TODO

	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) reconcileDelete(
	ctx context.Context,
	log logr.Logger,
	a *pipecdv1alpha1.Application,
	applicationService *services.ApplicationService,
) (ctrl.Result, error) {

	// delete Application
	if err := applicationService.Delete(ctx, a.Status.ApplicationId); err != nil {
		log.Error(err, err.Error())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
