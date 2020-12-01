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
	"github.com/ShotaKitazawa/pipecd-operator/pkg/mongo"
)

// MongoReconciler reconciles a Mongo object
type MongoReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *MongoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipecdv1alpha1.Mongo{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=mongoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pipecd.kanatakita.com,resources=mongoes/status,verbs=get;update;patch

func (r *MongoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("mongo", req.NamespacedName)

	/* Load Mongo */
	var m pipecdv1alpha1.Mongo
	log.Info("fetching Mongo Resource")
	if err := r.Get(ctx, req.NamespacedName, &m); err != nil {
		if errors_.IsNotFound(err) {
			r.Log.Info("Mongo not found", "Namespace", req.Namespace, "Name", req.Name)
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

	{ // mongodb
		wg.Add(1)
		errStreamMongoDB := make(chan error)
		go r.reconcileMongoDB(ctx, &wg, errStreamMongoDB, &m)
		select {
		case <-terminated:
		case err = <-errStreamMongoDB:
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *MongoReconciler) reconcileMongoDB(ctx context.Context, wg *sync.WaitGroup, errStream chan<- error, m *pipecdv1alpha1.Mongo) {
	defer (*wg).Done()
	log := r.Log.WithValues("component", "mongodb")
	mongodbNN := mongo.MakeMongoNamespacedName(m.Name, m.Namespace)

	/* Generate mongodb StatefulSet (NamespacedName) */
	mongodbStatefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mongodbNN.Name,
			Namespace: mongodbNN.Namespace,
		},
	}

	/* Apply mongodb StatefulSet */
	if _, err := ctrl.CreateOrUpdate(ctx, r, mongodbStatefulSet, func() (err error) {
		mongodbStatefulSet.Spec, err = mongo.MakeMongoStatefulSetSpec(*m)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(m, mongodbStatefulSet, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Mongo to StatefulSet")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure StatefulSet is correct")
		errStream <- err
		return
	}

	/* Generate mongodb Service (NamespacedName) */
	mongodbService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mongodbNN.Name,
			Namespace: mongodbNN.Namespace,
		},
	}

	/* Apply mongodb Service */
	if _, err := ctrl.CreateOrUpdate(ctx, r, mongodbService, func() (err error) {
		mongodbService.Spec, err = mongo.MakeMongoServiceSpec(*m)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(m, mongodbService, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Mongo to Service")
			return err
		}
		/* Get gateway Service from cluster */
		var mongodbServiceApplied v1.Service
		if err := r.Get(ctx, client.ObjectKey{Namespace: mongodbService.Namespace, Name: mongodbService.Name}, &mongodbServiceApplied); err != nil {
			// if does not exist, skip
			return nil
		}
		mongodbService.Spec.ClusterIP = mongodbServiceApplied.Spec.ClusterIP

		return nil
	}); err != nil {
		log.Error(err, "unable to ensure Service is correct")
		errStream <- err
		return
	}
}
