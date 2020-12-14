package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	webservicepb "github.com/pipe-cd/pipe/pkg/app/api/service/webservice"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	errors_ "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/object"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/pipecd"
)

const (
	environmentFinalizerName = "environment.finalizers.pipecd.kanatakita.com"
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
	log := r.Log.WithValues("environment", req.NamespacedName)

	/* Load Environment */
	var e pipecdv1alpha1.Environment
	log.Info("fetching Environment Resource")
	if err := r.Get(ctx, req.NamespacedName, &e); err != nil {
		if errors_.IsNotFound(err) {
			log.Info("Environment not found", "Namespace", req.Namespace, "Name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !e.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &e)
	}

	return r.reconcile(ctx, &e)
}

func (r *EnvironmentReconciler) reconcile(ctx context.Context, e *pipecdv1alpha1.Environment) (ctrl.Result, error) {

	/* Load Secret (key: encryption-key) */
	var secret v1.Secret
	if err := r.Get(ctx, types.NamespacedName{Namespace: e.Namespace, Name: e.Spec.EncryptionKeyRef.SecretName}, &secret); err != nil {
		return ctrl.Result{}, err
	}
	encryptionKeyBytes, ok := secret.Data[e.Spec.EncryptionKeyRef.Key]
	if !ok {
		return ctrl.Result{}, fmt.Errorf(`no key "%v" in Secret %v`, e.Spec.EncryptionKeyRef.Key, e.Spec.Name)
	}

	/* Load & Validate ControlPlaneList */
	var cpList pipecdv1alpha1.ControlPlaneList
	if err := r.List(ctx, &cpList, &client.ListOptions{Namespace: e.Namespace}); err != nil {
		r.Log.Info("ControlPlane not found", "Namespace", e.Namespace)
		return ctrl.Result{}, err
	} else if len(cpList.Items) != 1 {
		r.Log.Info("ControlPlane is none of more than 1", "Namespace", e.Namespace)
		return ctrl.Result{}, fmt.Errorf("ControlPlane more than 1: %v", cpList.Items)
	}
	cp := cpList.Items[0]

	serverNN := object.MakeServerNamespacedName(cp.Name, cp.Namespace)
	pipecdServerAddr := fmt.Sprintf("%s.%s.svc:%d",
		serverNN.Name,
		serverNN.Namespace,
		object.ServerContainerWebServerPort,
	)

	client, ctx, err := pipecd.NewClientGenerator(string(encryptionKeyBytes), e.Spec.ProjectID, e.Spec.Insecure).
		GeneratePipeCdWebServiceClient(ctx, pipecdServerAddr)
	if err != nil {
		return ctrl.Result{}, err
	}

	// list PipeCD Environment & return if Environment already exists
	resp, err := client.ListEnvironments(ctx, &webservicepb.ListEnvironmentsRequest{})
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, env := range resp.Environments {
		if e.Spec.Name == env.Name {
			return ctrl.Result{}, nil
		}
	}

	// add PipeCD Environment
	if _, err := client.AddEnvironment(ctx, &webservicepb.AddEnvironmentRequest{Name: e.Spec.Name, Desc: e.Spec.Description}); err != nil {
		return ctrl.Result{}, err
	}

	// update Environment Object
	e.ObjectMeta.Finalizers = append(e.ObjectMeta.Finalizers, environmentFinalizerName)
	if err := r.Update(context.Background(), e); err != nil {
		return ctrl.Result{}, err
	}

	// record to event
	r.Recorder.Eventf(e, corev1.EventTypeNormal, "Updated", "Update Environment.Metadata.Finalizer")

	return ctrl.Result{}, nil
}

func (r *EnvironmentReconciler) reconcileDelete(ctx context.Context, e *pipecdv1alpha1.Environment) (ctrl.Result, error) {

	// TBD: PipeCD v0.9.0, no method DELETE of PipeCD Environment.

	// remove Finalizer of Environment Object
	var tmp []string
	for _, val := range e.ObjectMeta.Finalizers {
		if val != environmentFinalizerName {
			tmp = append(tmp, val)
		}
	}
	e.ObjectMeta.Finalizers = tmp
	if err := r.Update(context.Background(), e); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
