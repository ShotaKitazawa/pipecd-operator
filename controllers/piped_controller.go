package controllers

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/golang/protobuf/ptypes/wrappers"
	webservicepb "github.com/pipe-cd/pipe/pkg/app/api/service/webservice"
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

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/object"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/pipecd"
)

const (
	pipedFinalizerName = "piped.finalizers.pipecd.kanatakita.com"
)

var (
	pipedKeyCache           string
	pipedKeyCacheMutex      sync.Mutex
	encryptionKeyCache      string
	encryptionKeyCacheMutex sync.Mutex
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
	log := r.Log.WithValues("namespacedName", req.NamespacedName)

	/* Load Piped */
	var p pipecdv1alpha1.Piped
	log.Info("fetching Piped Resource")
	if err := r.Get(ctx, req.NamespacedName, &p); err != nil {
		if errors_.IsNotFound(err) {
			log.Info("Piped not found", "Namespace", req.Namespace, "Name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !p.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, &p)
	}

	return r.reconcile(ctx, &p)
}

func (r *PipedReconciler) reconcile(ctx context.Context, p *pipecdv1alpha1.Piped) (ctrl.Result, error) {
	/* Load Secret (key: encryption-key) */
	if encryptionKeyCache == "" {
		var secret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{Namespace: p.Namespace, Name: p.Spec.EncryptionKeyRef.SecretName}, &secret); err != nil {
			return ctrl.Result{}, err
		}
		encryptionKeyBytes, ok := secret.Data[p.Spec.EncryptionKeyRef.Key]
		if !ok {
			return ctrl.Result{}, fmt.Errorf(`no key "%v" in Secret %v`, p.Spec.EncryptionKeyRef.Key, p.Spec.Name)
		}
		encryptionKeyCacheMutex.Lock()
		encryptionKeyCache = string(encryptionKeyBytes)
		encryptionKeyCacheMutex.Unlock()
	}

	/* Load & Validate ControlPlaneList */
	var cpList pipecdv1alpha1.ControlPlaneList
	if err := r.List(ctx, &cpList, &client.ListOptions{Namespace: p.Namespace}); err != nil {
		r.Log.Info("Piped not found", "Namespace", p.Namespace)
		return ctrl.Result{}, err
	} else if len(cpList.Items) != 1 {
		r.Log.Info("Piped is none of more than 1", "Namespace", p.Namespace)
		return ctrl.Result{}, fmt.Errorf("Piped is none or more than 1: %v", cpList.Items)
	}
	cp := cpList.Items[0]
	serverNN := object.MakeServerNamespacedName(cp.Name, cp.Namespace)
	pipecdServerAddr := fmt.Sprintf("%s.%s.svc:%d",
		serverNN.Name,
		serverNN.Namespace,
		object.ServerContainerWebServerPort,
	)

	// generate gRPC Client for Piped
	client, ctx, err := pipecd.NewClientGenerator(encryptionKeyCache, p.Spec.ProjectID, p.Spec.Insecure).
		GeneratePipeCdWebServiceClient(ctx, pipecdServerAddr)
	if err != nil {
		return ctrl.Result{}, err
	}

	// list PipeCD Piped & return if Piped already exists
	resp, err := client.ListPipeds(ctx, &webservicepb.ListPipedsRequest{Options: &webservicepb.ListPipedsRequest_Options{Enabled: &wrappers.BoolValue{Value: true}}})
	if err != nil {
		return ctrl.Result{}, err
	}
	var pipedId string
	for _, piped := range resp.Pipeds {
		if p.Spec.Name == piped.Name {
			pipedId = piped.Id
		}
	}

	if pipedId == "" {
		// list PipeCD Environments & return if Environment does not exist
		respListEnvironments, err := client.ListEnvironments(ctx, &webservicepb.ListEnvironmentsRequest{})
		if err != nil {
			return ctrl.Result{}, err
		}

		// get Environments ID
		var envIds []string
		for _, env := range respListEnvironments.Environments {
			if env.Name == p.Spec.Environment {
				envIds = append(envIds, env.Id)
			}
		}
		if len(envIds) == 0 {
			return ctrl.Result{}, fmt.Errorf("Environment %v does not exist", p.Spec.Name)
		}

		// register PipeCD Piped
		respRegisterPiped, err := client.RegisterPiped(ctx, &webservicepb.RegisterPipedRequest{
			Name:   p.Spec.Name,
			Desc:   p.Spec.Description,
			EnvIds: envIds,
		})
		if err != nil {
			return ctrl.Result{}, err
		}
		pipedId = respRegisterPiped.Id
		pipedKeyCacheMutex.Lock()
		pipedKeyCache = respRegisterPiped.Key
		pipedKeyCacheMutex.Unlock()
	}

	// generate K8s Objects for Piped
	if pipedKeyCache == "" {
		err := fmt.Errorf("pipedKeyCache is none. Please disable piped configuration from WebUI.")
		r.Log.Error(err, err.Error())
		return ctrl.Result{}, err
	}
	if result, err := r.reconcilePiped(ctx, p, pipedId, pipedKeyCache); err != nil {
		return result, err
	}

	// update Piped Object
	var finalizerExists bool
	for _, finalizer := range p.ObjectMeta.Finalizers {
		if finalizer == pipedFinalizerName {
			finalizerExists = true
		}
	}
	if !finalizerExists {
		p.ObjectMeta.Finalizers = append(p.ObjectMeta.Finalizers, pipedFinalizerName)
		if err := r.Update(ctx, p); err != nil {
			return ctrl.Result{}, err
		}
		// record to event
		r.Recorder.Eventf(p, corev1.EventTypeNormal, "Updated", "Update Piped.Metadata.Finalizer")
	}

	return ctrl.Result{}, nil
}

func (r *PipedReconciler) reconcilePiped(ctx context.Context, p *pipecdv1alpha1.Piped, pipedId, pipedKey string) (ctrl.Result, error) {
	log := r.Log.WithValues("component", "piped")
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
		pipedConfigMap.BinaryData, err = object.MakePipedConfigMapData(*p, pipedId)
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
		log.Error(err, "unable to ensure deployment is correct")
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
		log.Error(err, "unable to ensure deployment is correct")
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
		pipedDeployment.Spec, err = object.MakePipedDeploymentSpec(*p)
		if err != nil {
			return err
		}
		if err := ctrl.SetControllerReference(p, pipedDeployment, r.Scheme); err != nil {
			log.Error(err, "unable to set ownerReference from Piped to Deployment")
			return err
		}
		return nil
	}); err != nil {
		log.Error(err, "unable to ensure deployment is correct")
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
	if availableReplicas != p.Status.AvailableReplicas {
		p.Status.AvailableReplicas = availableReplicas
		if err := r.Status().Update(ctx, p); err != nil {
			log.Error(err, "unable to update Piped status")
			return ctrl.Result{}, err
		}
		/* Record to event */
		r.Recorder.Eventf(p, corev1.EventTypeNormal, "Updated", "Update Piped.status.availablePipedReplicas: %d", p.Status.AvailableReplicas)
	}
	if pipedId != p.Status.PipedId {
		p.Status.PipedId = pipedId
		if err := r.Status().Update(ctx, p); err != nil {
			log.Error(err, "unable to update Piped status")
			return ctrl.Result{}, err
		}
		/* Record to event */
		r.Recorder.Eventf(p, corev1.EventTypeNormal, "Updated", "Update Piped.status.pipedId: %d", p.Status.AvailableReplicas)
	}

	return ctrl.Result{}, nil
}

func (r *PipedReconciler) reconcileDelete(ctx context.Context, p *pipecdv1alpha1.Piped) (ctrl.Result, error) {
	/* Load Secret (key: encryption-key) */
	if encryptionKeyCache == "" {
		var secret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{Namespace: p.Namespace, Name: p.Spec.EncryptionKeyRef.SecretName}, &secret); err != nil {
			return ctrl.Result{}, err
		}
		encryptionKeyBytes, ok := secret.Data[p.Spec.EncryptionKeyRef.Key]
		if !ok {
			return ctrl.Result{}, fmt.Errorf(`no key "%v" in Secret %v`, p.Spec.EncryptionKeyRef.Key, p.Spec.Name)
		}
		encryptionKeyCacheMutex.Lock()
		encryptionKeyCache = string(encryptionKeyBytes)
		encryptionKeyCacheMutex.Unlock()
	}

	/* Load & Validate ControlPlaneList */
	var cpList pipecdv1alpha1.ControlPlaneList
	if err := r.List(ctx, &cpList, &client.ListOptions{Namespace: p.Namespace}); err != nil {
		r.Log.Info("Piped not found", "Namespace", p.Namespace)
		return ctrl.Result{}, err
	} else if len(cpList.Items) != 1 {
		r.Log.Info("Piped is none or more than 1", "Namespace", p.Namespace)
		return ctrl.Result{}, fmt.Errorf("Piped is none or more than 1: %v", cpList.Items)
	}
	cp := cpList.Items[0]
	serverNN := object.MakeServerNamespacedName(cp.Name, cp.Namespace)
	pipecdServerAddr := fmt.Sprintf("%s.%s.svc:%d",
		serverNN.Name,
		serverNN.Namespace,
		object.ServerContainerWebServerPort,
	)

	// generate gRPC Client for Piped
	client, ctx, err := pipecd.NewClientGenerator(encryptionKeyCache, p.Spec.ProjectID, p.Spec.Insecure).
		GeneratePipeCdWebServiceClient(ctx, pipecdServerAddr)
	if err != nil {
		return ctrl.Result{}, err
	}

	// get PipedID from Piped.status.pipedID
	pipedId := p.Status.PipedId

	// disable Piped
	if _, err := client.DisablePiped(ctx, &webservicepb.DisablePipedRequest{PipedId: pipedId}); err != nil {
		return ctrl.Result{}, err
	}

	// remove Finalizer of Piped Object
	var tmp []string
	for _, val := range p.ObjectMeta.Finalizers {
		if val != pipedFinalizerName {
			tmp = append(tmp, val)
		}
	}
	p.ObjectMeta.Finalizers = tmp
	if err := r.Update(context.Background(), p); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
