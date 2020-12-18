package v1alpha1

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipedSpec defines the desired state of Piped
type PipedSpec struct {
	// +kubebuilder:validation:Required

	Version string `json:"version"`

	// +kubebuilder:validation:Required

	Name string `json:"name"`

	// +kubebuilder:validation:Required

	Description string `json:"description"`

	// +kubebuilder:validation:Required

	ProjectId string `json:"projectID"`

	// +kubebuilder:validation:Required

	EnvironmentRef EnvironmentRef `json:"environmentRef"`

	// +kubebuilder:validation:Optional

	Insecure bool `json:"insecure"`

	// +kubebuilder:validation:Required

	// Config is Piped configuration of PipeCD
	Config PipedConfigSpec `json:"config"`
}

type PipedConfig struct {
	Kind       string          `json:"kind"`
	APIVersion string          `json:"apiversion"`
	Spec       PipedConfigSpec `json:"spec"`
}

type PipedConfigSpec struct {

	// +kubebuilder:validation:Optional

	// The identifier of the PipeCD project where this piped belongs to.
	ProjectID string `json:"projectID"`

	// +kubebuilder:validation:Optional

	// The generated ID for this piped.
	PipedID string `json:"pipedID"`

	// +kubebuilder:validation:Optional

	// The path to the file containing the generated Key string for this piped.
	PipedKeyFile string `json:"pipedKeyFile"`

	// +kubebuilder:validation:Required

	// The address used to connect to the control-plane's API.
	APIAddress string `json:"apiAddress"`

	// +kubebuilder:validation:Required

	// The address to the control-plane's Web.
	WebAddress string `json:"webAddress"`

	// +kubebuilder:validation:Optional
	// TODO: apiextensions.k8s.io/v1 // +kubebuilder:default:=60

	// How often to check whether an application should be synced.
	// Default is 1m.
	SyncInterval int64 `json:"syncInterval"`

	// +kubebuilder:validation:Optional

	// Git configuration needed for git commands.
	Git PipedGit `json:"git,omitempty"`

	// +kubebuilder:validation:Optional

	// List of git repositories this piped will handle.
	Repositories []PipedRepository `json:"repositories"`

	// +kubebuilder:validation:Optional

	// List of helm chart repositories that should be added while starting up.
	ChartRepositories []HelmChartRepository `json:"chartRepositories,omitempty"`

	// +kubebuilder:validation:Optional

	// List of cloud providers can be used by this piped.
	CloudProviders []PipedCloudProvider `json:"cloudProviders,omitempty"`

	// +kubebuilder:validation:Optional

	// List of analysis providers can be used by this piped.
	AnalysisProviders []PipedAnalysisProvider `json:"analysisProviders,omitempty"`

	// +kubebuilder:validation:Optional

	// List of image providers can be used by this piped.
	// TODO
	//ImageProviders []PipedImageProvider `json:"imageProviders"`

	// +kubebuilder:validation:Optional

	// Sending notification to Slack, Webhook…
	Notifications Notifications `json:"notifications"`

	// +kubebuilder:validation:Optional

	// How the sealed secret should be managed.
	// TODO
	//SealedSecretManagement *SealedSecretManagement `json:"sealedSecretManagement"`
}

type PipedGit struct {

	// +kubebuilder:validation:Optional

	// The username that will be configured for `git` user.
	Username string `json:"username,omitempty"`

	// +kubebuilder:validation:Optional

	// The email that will be configured for `git` user.
	Email string `json:"email,omitempty"`

	// +kubebuilder:validation:Optional

	// Where to write ssh config file.
	// Default is "/home/pipecd/.ssh/config".
	SSHConfigFilePath string `json:"sshConfigFilePath,omitempty"`

	// +kubebuilder:validation:Optional

	// The host name.
	// e.g. github.com, gitlab.com
	// Default is "github.com".
	Host string `json:"host,omitempty"`

	// +kubebuilder:validation:Optional

	// The hostname or IP address of the remote git server.
	// e.g. github.com, gitlab.com
	// Default is the same value with Host.
	HostName string `json:"hostName,omitempty"`

	// +kubebuilder:validation:Optional

	// The path to the private ssh key file.
	// This will be used to clone the source code of the specified git repositories.
	SSHKeyFile string `json:"sshKeyFile,omitempty"`
}

type PipedRepository struct {

	// +kubebuilder:validation:Optional

	// Unique identifier for this repository.
	// This must be unique in the piped scope.
	RepoID string `json:"repoId"`

	// +kubebuilder:validation:Optional

	// Remote address of the repository used to clone the source code.
	// e.g. git@github.com:org/repo.git
	Remote string `json:"remote"`

	// +kubebuilder:validation:Optional

	// The branch will be handled.
	Branch string `json:"branch"`
}

type HelmChartRepository struct {

	// +kubebuilder:validation:Optional

	// The name of the Helm chart repository.
	Name string `json:"name"`

	// +kubebuilder:validation:Optional

	// The address to the Helm chart repository.
	Address string `json:"address"`

	// +kubebuilder:validation:Optional

	// Username used for the repository backed by HTTP basic authentication.
	Username string `json:"username"`

	// +kubebuilder:validation:Optional

	// Password used for the repository backed by HTTP basic authentication.
	Password string `json:"password"`
}

type PipedCloudProviderType string

const (
	PipedCloudProviderTypeKubernetes PipedCloudProviderType = "KUBERNETES"
	PipedCloudProviderTypeTerraform  PipedCloudProviderType = "TERRAFORM"
	PipedCloudProviderTypeCloudRun   PipedCloudProviderType = "CLOUDRUN"
	PipedCloudProviderTypeLambda     PipedCloudProviderType = "LAMBDA"
)

type genericPipedCloudProvider struct {
	Name   string                 `json:"name"`
	Type   PipedCloudProviderType `json:"type"`
	Config json.RawMessage        `json:"config"`
}

type PipedCloudProvider struct {

	// +kubebuilder:validation:Required

	// The name of the cloud provider.
	Name string `json:"name"`

	// +kubebuilder:validation:Optional

	// config of Kubernetes
	KubernetesConfig *CloudProviderKubernetesConfig `json:"kubernetesConfig"`

	// +kubebuilder:validation:Optional

	// config of Terraform
	TerraformConfig *CloudProviderTerraformConfig `json:"terraformConfig"`

	// +kubebuilder:validation:Optional

	// config of CloudRun
	CloudRunConfig *CloudProviderCloudRunConfig `json:"cloudRunConfig"`

	// +kubebuilder:validation:Optional

	// config of Lambda
	LambdaConfig *CloudProviderLambdaConfig `json:"lambdaConfig"`
}

func (p *PipedCloudProvider) MarshalJSON() ([]byte, error) {
	var data []byte
	var err error
	gp := genericPipedCloudProvider{Name: p.Name}

	// TODO: this logic is "or", but should be "xor"
	switch {
	case p.KubernetesConfig != nil:
		gp.Type = PipedCloudProviderTypeKubernetes
		data, err = json.Marshal(p.KubernetesConfig)
		if err != nil {
			return nil, err
		}
	case p.TerraformConfig != nil:
		gp.Type = PipedCloudProviderTypeTerraform
		data, err = json.Marshal(p.TerraformConfig)
		if err != nil {
			return nil, err
		}
	case p.CloudRunConfig != nil:
		gp.Type = PipedCloudProviderTypeCloudRun
		data, err = json.Marshal(p.CloudRunConfig)
		if err != nil {
			return nil, err
		}
	case p.LambdaConfig != nil:
		gp.Type = PipedCloudProviderTypeLambda
		data, err = json.Marshal(p.LambdaConfig)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid Config")
	}
	if err := json.Unmarshal(data, &gp.Config); err != nil {
		return nil, err
	}
	return json.Marshal(&gp)
}

type CloudProviderKubernetesConfig struct {

	// +kubebuilder:validation:Optional

	// The master URL of the kubernetes cluster.
	// Empty means in-cluster.
	MasterURL string `json:"masterURL"`

	// +kubebuilder:validation:Optional

	// The path to the kubeconfig file.
	// Empty means in-cluster.
	KubeConfigPath string `json:"kubeConfigPath"`

	// +kubebuilder:validation:Optional

	// Configuration for application resource informer.
	AppStateInformer KubernetesAppStateInformer `json:"appStateInformer"`
}

type KubernetesAppStateInformer struct {

	// +kubebuilder:validation:Optional

	// Only watches the specified namespace.
	// Empty means watching all namespaces.
	Namespace string `json:"namespace"`

	// +kubebuilder:validation:Optional

	// List of resources that should be added to the watching targets.
	IncludeResources []KubernetesResourceMatcher `json:"includeResources"`

	// +kubebuilder:validation:Optional

	// List of resources that should be ignored from the watching targets.
	ExcludeResources []KubernetesResourceMatcher `json:"excludeResources"`
}

type KubernetesResourceMatcher struct {

	// +kubebuilder:validation:Required

	// The APIVersion of the kubernetes resource.
	APIVersion string `json:"apiVersion"`

	// +kubebuilder:validation:Optional

	// The kind name of the kubernetes resource.
	// Empty means all kinds are matching.
	Kind string `json:"kind"`
}

type CloudProviderTerraformConfig struct {

	// +kubebuilder:validation:Optional

	// List of variables that will be set directly on terraform commands with "-var" flag.
	// The variable must be formatted by "key=value" as below:
	// "image_id=ami-abc123"
	// 'image_id_list=["ami-abc123","ami-def456"]'
	// 'image_id_map={"us-east-1":"ami-abc123","us-east-2":"ami-def456"}'
	Vars []string `json:"vars"`
}

type CloudProviderCloudRunConfig struct {

	// +kubebuilder:validation:Required

	// The GCP project hosting the CloudRun service.
	Project string `json:"project"`

	// +kubebuilder:validation:Required

	// The region of running CloudRun service.
	Region string `json:"region"`

	// +kubebuilder:validation:Optional

	// The path to the service account file for accessing CloudRun service.
	CredentialsFile string `json:"credentialsFile"`
}

type CloudProviderLambdaConfig struct {

	// +kubebuilder:validation:Required

	Region string `json:"region"`
}

type PipedAnalysisProviderType string

const (
	PipedAnalysisProviderTypePrometheus  PipedAnalysisProviderType = "PROMETHEUS"
	PipedAnalysisProviderTypeDatadog     PipedAnalysisProviderType = "DATADOG"
	PipedAnalysisProviderTypeStackdriver PipedAnalysisProviderType = "STACKDRIVER"
)

type genericPipedAnalysisProvider struct {
	Name   string                    `json:"name"`
	Type   PipedAnalysisProviderType `json:"type"`
	Config json.RawMessage           `json:"config"`
}

type PipedAnalysisProvider struct {
	// +kubebuilder:validation:Required

	// The name of the analysis provider.
	Name string `json:"name"`

	// +kubebuilder:validation:Optional

	// Config of Prometheus
	PrometheusConfig *AnalysisProviderPrometheusConfig `json:"prometheus"`

	// +kubebuilder:validation:Optional

	// Config of Datadog
	DatadogConfig *AnalysisProviderDatadogConfig `json:"datadog"`

	// +kubebuilder:validation:Optional

	// Config of Stackdriver
	StackdriverConfig *AnalysisProviderStackdriverConfig `json:"stackdriver"`
}

func (p *PipedAnalysisProvider) MarshalJSON() ([]byte, error) {
	var data []byte
	var err error
	gp := genericPipedAnalysisProvider{Name: p.Name}

	// TODO: this logic is "or", but should be "xor"
	switch {
	case p.PrometheusConfig != nil:
		gp.Type = PipedAnalysisProviderTypePrometheus
		data, err = json.Marshal(p.PrometheusConfig)
		if err != nil {
			return nil, err
		}
	case p.DatadogConfig != nil:
		gp.Type = PipedAnalysisProviderTypeDatadog
		data, err = json.Marshal(p.DatadogConfig)
		if err != nil {
			return nil, err
		}
	case p.StackdriverConfig != nil:
		gp.Type = PipedAnalysisProviderTypeStackdriver
		data, err = json.Marshal(p.StackdriverConfig)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid Config")
	}
	if err := json.Unmarshal(data, &gp.Config); err != nil {
		return nil, err
	}
	return json.Marshal(&gp)
}

type AnalysisProviderPrometheusConfig struct {

	// +kubebuilder:validation:Required

	Address string `json:"address"`

	// +kubebuilder:validation:Optional

	// The path to the username file.
	UsernameFile string `json:"usernameFile"`

	// +kubebuilder:validation:Optional

	// The path to the password file.
	PasswordFile string `json:"passwordFile"`
}

type AnalysisProviderDatadogConfig struct {

	// +kubebuilder:validation:Required

	Address string `json:"address"`

	// +kubebuilder:validation:Required

	// The path to the api key file.
	APIKeyFile string `json:"apiKeyFile"`

	// +kubebuilder:validation:Required

	// The path to the application key file.
	ApplicationKeyFile string `json:"applicationKeyFile"`
}

type AnalysisProviderStackdriverConfig struct {

	// +kubebuilder:validation:Required

	// The path to the service account file.
	ServiceAccountFile string `json:"serviceAccountFile"`
}

type Notifications struct {

	// +kubebuilder:validation:Optional

	// List of notification routes.
	Routes []NotificationRoute `json:"routes,omitempty"`

	// +kubebuilder:validation:Optional

	// List of notification receivers.
	Receivers []NotificationReceiver `json:"receivers,omitempty"`
}

type NotificationRoute struct {

	// +kubebuilder:validation:Required

	// The name of the route.
	Name string `json:"name"`

	// +kubebuilder:validation:Required

	// The name of receiver who will receive all matched events.
	Receiver string `json:"receiver"`

	// +kubebuilder:validation:Optional

	// List of events that should be routed to the receiver.
	Events []string `json:"events"`

	// +kubebuilder:validation:Optional

	// List of events that should be ignored.
	IgnoreEvents []string `json:"ignoreEvents"`

	// +kubebuilder:validation:Optional

	// List of event groups should be routed to the receiver.
	Groups []string `json:"groups"`

	// +kubebuilder:validation:Optional

	// List of event groups should be ignored.
	IgnoreGroups []string `json:"ignoreGroups"`

	// +kubebuilder:validation:Optional

	// List of applications where their events should be routed to the receiver.
	Apps []string `json:"apps"`

	// +kubebuilder:validation:Optional

	// List of applications where their events should be ignored.
	IgnoreApps []string `json:"ignoreApps"`

	// +kubebuilder:validation:Optional

	// List of environments where their events should be routed to the receiver.
	Envs []string `json:"envs"`

	// +kubebuilder:validation:Optional

	// List of environments where their events should be ignored.
	IgnoreEnvs []string `json:"ignoreEnvs"`
}

type NotificationReceiver struct {

	// +kubebuilder:validation:Required

	// The name of the receiver.
	Name string `json:"name"`

	// +kubebuilder:validation:Optional

	// Configuration for slack receiver.
	Slack *NotificationReceiverSlack `json:"slack"`

	// +kubebuilder:validation:Optional

	// Configuration for webhook receiver.
	Webhook *NotificationReceiverWebhook `json:"webhook"`
}

type NotificationReceiverSlack struct {

	// +kubebuilder:validation:Required

	// The hookURL of a slack channel.
	HookURL string `json:"hookURL"`
}

type NotificationReceiverWebhook struct {

	// +kubebuilder:validation:Required

	URL string `json:"url"`
}

// PipedStatus defines the observed state of Piped
type PipedStatus struct {
	// +kubebuilder:validation:Optional

	// this is equal deployment.status.availableReplicas of piped
	AvailableReplicas int32 `json:"availableReplicas"`

	// +kubebuilder:validation:Optional

	// Environment ID
	EnvironmentIds string `json:"environmentID"`

	// +kubebuilder:validation:Optional

	// Piped ID
	PipedId string `json:"pipedID"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Piped is the Schema for the pipeds API
type Piped struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipedSpec   `json:"spec,omitempty"`
	Status PipedStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PipedList contains a list of Piped
type PipedList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Piped `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Piped{}, &PipedList{})
}
