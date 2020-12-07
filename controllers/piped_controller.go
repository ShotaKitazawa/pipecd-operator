package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
)

// PipedReconciler reconciles a Piped object
type PipedReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=pipeds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=pipeds/status,verbs=get;update;patch

func (r *PipedReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("piped", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *PipedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipecdv1alpha1.Piped{}).
		Complete(r)
}
