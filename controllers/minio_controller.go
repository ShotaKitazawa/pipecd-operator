package controllers

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	errors_ "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/object"
)

// MinioReconciler reconciles a Minio object
type MinioReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *MinioReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipecdv1alpha1.Minio{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=minios,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=minios/status,verbs=get;update;patch

func (r *MinioReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("minio", req.NamespacedName)

	/* Load Minio */
	var m pipecdv1alpha1.Minio
	log.Info("fetching Minio Resource")
	if err := r.Get(ctx, req.NamespacedName, &m); err != nil {
		if errors_.IsNotFound(err) {
			r.Log.Info("Minio not found", "Namespace", req.Namespace, "Name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var err error
	var wg sync.WaitGroup
	terminated := make(chan interface{})
	go func() {
		wg.Wait()
		close(terminated)
	}()

	{ // minio
		wg.Add(1)
		errStreamMinio := make(chan error)
		go r.reconcileMinio(ctx, &wg, errStreamMinio, &m)
		select {
		case <-terminated:
		case err = <-errStreamMinio:
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *MinioReconciler) reconcileMinio(ctx context.Context, wg *sync.WaitGroup, errStream chan<- error, m *pipecdv1alpha1.Minio) {
	defer (*wg).Done()
	log := r.Log.WithValues("component", "minio")
	minioNN := object.MakeMinioNamespacedName(m.Name, m.Namespace)

	/* Generate minio StatefulSet (NamespacedName) */
	minioStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      minioNN.Name,
			Namespace: minioNN.Namespace,
		},
	}

	/* Apply minio StatefulSet */
	if _, err := ctrl.CreateOrUpdate(ctx, r, minioStatefulSet, func() (err error) {
		minioStatefulSet.Spec, err = object.MakeMinioStatefulSetSpec(*m)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(m, minioStatefulSet, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Minio to StatefulSet")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure deployment is correct")
		errStream <- err
		return
	}

	/* Generate minio Service (NamespacedName) */
	minioService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      minioNN.Name,
			Namespace: minioNN.Namespace,
		},
	}

	/* Apply minio Service */
	if _, err := ctrl.CreateOrUpdate(ctx, r, minioService, func() (err error) {
		minioService.Spec, err = object.MakeMinioServiceSpec(*m)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(m, minioService, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Minio to Service")
			return err
		}
		/* Get gateway Service from cluster */
		var minioServiceApplied v1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: minioService.Namespace, Name: minioService.Name}, &minioServiceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		minioService.Spec.ClusterIP = minioServiceApplied.Spec.ClusterIP

		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		errStream <- err
		return
	}
}
