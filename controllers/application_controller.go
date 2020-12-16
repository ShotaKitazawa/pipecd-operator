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
	var e pipecdv1alpha1.Application
	log.Info("fetching Application Resource")
	if err := r.Get(ctx, req.NamespacedName, &e); err != nil {
		if errors_.IsNotFound(err) {
			log.Info("Application not found", "Namespace", req.Namespace, "Name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !e.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &e)
	}

	return r.reconcile(ctx, &e)
}

func (r *ApplicationReconciler) reconcile(ctx context.Context, a *pipecdv1alpha1.Application) (ctrl.Result, error) {
	// TODO

	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) reconcileDelete(ctx context.Context, e *pipecdv1alpha1.Application) (ctrl.Result, error) {
	// TODO

	return ctrl.Result{}, nil
}
