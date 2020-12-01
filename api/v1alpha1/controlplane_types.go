package v1alpha1

import (
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ControlPlaneSpec defines all configuration for all control-plane components.
type ControlPlaneSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0

	// number of replicas of gateway
	ReplicasGateway *int32 `json:"replicasGateway,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0

	// number of replicas of api
	ReplicasApi *int32 `json:"replicasApi,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0

	// number of replicas of web
	ReplicasWeb *int32 `json:"replicasWeb,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0

	// number of replicas of cache
	ReplicasCache *int32 `json:"replicasCache,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0

	// number of replicas of ops
	ReplicasOps *int32 `json:"replicasOps,omitempty"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format:=string

	// version of gcr.io/pipecd/api & gcr.io/pipecd/web & gcr.io/pipecd/ops
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// version of github.com/envoyproxy/envoy
	EnvoyVersion string `json:"envoyVersion"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// version of docker.io/redis
	RedisVersion string `json:"redisVersion"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// version of docker.io/minio/minio
	MinioVersion string `json:"minioVersion"`

	// secret for PipeCD EncryptionKey & Minio AccessKey and SecretKey
	Secret *v1.SecretVolumeSource `json:"secret"`

	// +kubebuilder:validation:Optional

	// Config is ControlPlane configuration of PipeCD
	Config ControlPlaneConfigSpec `json:"config"`
}

type ControlPlaneConfig struct {
	Kind       string                 `json:"kind"`
	APIVersion string                 `json:"apiversion"`
	Spec       ControlPlaneConfigSpec `json:"spec"`
}

type ControlPlaneConfigSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// The address to the control plane.
	// This is required if SSO is enabled.
	Address string `json:"address,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// A randomly generated string used to sign oauth state.
	StateKey string `json:"stateKey,omitempty"`

	// +kubebuilder:validation:Optional

	// The configuration of datastore for control plane
	Datastore ControlPlaneDataStore `json:"datastore,omitempty"`

	// +kubebuilder:validation:Optional

	// The configuration of filestore for control plane.
	Filestore ControlPlaneFileStore `json:"filestore,omitempty"`

	// +kubebuilder:validation:Optional

	// The configuration of cache for control plane.
	Cache ControlPlaneCache `json:"cache,omitempty"`

	// +kubebuilder:validation:Optional

	// List of debugging/quickstart projects defined in Control Plane configuration.
	// Please note that do not use this to configure the projects running in the production.
	Projects []ControlPlaneProject `json:"projects,omitempty"`

	// +kubebuilder:validation:Optional

	// List of shared SSO configurations that can be used by any projects.
	SharedSSOConfigs []SharedSSOConfig `json:"sharedSSOConfigs,omitempty"`
}

type ControlPlaneDataStoreType string

const (
	ControlPlaneDataStoreTypeFirestore ControlPlaneDataStoreType = "FILESTORE"
	ControlPlaneDataStoreTypeDynamoDB  ControlPlaneDataStoreType = "DYNAMODB"
	ControlPlaneDataStoreTypeMongoDB   ControlPlaneDataStoreType = "MONGODB"
)

type genericControlPlaneDataStore struct {
	Type   ControlPlaneDataStoreType `json:"type"`
	Config json.RawMessage           `json:"config"`
}

type ControlPlaneDataStore struct {
	// +kubebuilder:validation:Optional

	// The configuration in the case of Cloud Firestore.
	FirestoreConfig *DataStoreFireStoreConfig `json:"firestoreConfig"`

	// +kubebuilder:validation:Optional

	// The configuration in the case of Amazon DynamoDB.
	DynamoDBConfig *DataStoreDynamoDBConfig `json:"dynamoDbConfig"`

	// +kubebuilder:validation:Optional

	// The configuration in the case of general MongoDB.
	MongoDBConfig *DataStoreMongoDBConfig `json:"mongoDBConfig"`
}

func (d *ControlPlaneDataStore) MarshalJSON() ([]byte, error) {
	var data []byte
	var err error
	gd := genericControlPlaneDataStore{}

	// TODO: this logic is "or", but should be "xor"
	switch {
	case d.FirestoreConfig != nil:
		gd.Type = ControlPlaneDataStoreTypeFirestore
		data, err = json.Marshal(d.FirestoreConfig)
		if err != nil {
			return nil, err
		}
	case d.DynamoDBConfig != nil:
		gd.Type = ControlPlaneDataStoreTypeDynamoDB
		data, err = json.Marshal(d.DynamoDBConfig)
		if err != nil {
			return nil, err
		}
	case d.MongoDBConfig != nil:
		gd.Type = ControlPlaneDataStoreTypeMongoDB
		data, err = json.Marshal(d.MongoDBConfig)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid Config")
	}
	if err := json.Unmarshal(data, &gd.Config); err != nil {
		return nil, err
	}
	return json.Marshal(&gd)
}

// TBD
type DataStoreFireStoreConfig struct {
}

// TBD
type DataStoreDynamoDBConfig struct {
}

type DataStoreMongoDBConfig struct {

	// +kubebuilder:validation:Format:=string

	// The url of MongoDB. All of credentials can be specified via this field.
	URL string `json:"url"`

	// +kubebuilder:validation:Format:=string

	// The name of the database.
	Database string `json:"database"`

	// +kubebuilder:validation:Format:=string
	// +kubebuilder:validation:Optional

	// The path to the username file.
	// For those who don't want to include the username in the URL.
	UsernameFile string `json:"usernameFile"`

	// +kubebuilder:validation:Format:=string
	// +kubebuilder:validation:Optional

	// The path to the password file.
	// For those who don't want to include the password in the URL.
	PasswordFile string `json:"passwordFile"`
}

type ControlPlaneFileStoreType string

const (
	ControlPlaneFileStoreTypeGCS   ControlPlaneFileStoreType = "GCS"
	ControlPlaneFileStoreTypeS3    ControlPlaneFileStoreType = "S3"
	ControlPlaneFileStoreTypeMinio ControlPlaneFileStoreType = "MINIO"
)

type genericControlPlaneFileStore struct {
	Type   ControlPlaneFileStoreType `json:"type"`
	Config json.RawMessage           `json:"config"`
}

type ControlPlaneFileStore struct {
	// +kubebuilder:validation:Optional

	// The configuration in the case of Google Cloud Storage.
	GCSConfig *FileStoreGCSConfig `json:"gcsConfig"`

	// +kubebuilder:validation:Optional

	// The configuration in the case of Amazon S3.
	S3Config *FileStoreS3Config `json:"s3Config"`

	// +kubebuilder:validation:Optional

	// The configuration in the case of Minio.
	MinioConfig *FileStoreMinioConfig `json:"minioConfig"`
}

func (f *ControlPlaneFileStore) MarshalJSON() ([]byte, error) {
	var data []byte
	var err error
	gf := genericControlPlaneFileStore{}

	// TODO: this logic is "or", but should be "xor"
	switch {
	case f.GCSConfig != nil:
		gf.Type = ControlPlaneFileStoreTypeGCS
		data, err = json.Marshal(f.GCSConfig)
		if err != nil {
			return nil, err
		}
	case f.S3Config != nil:
		gf.Type = ControlPlaneFileStoreTypeS3
		data, err = json.Marshal(f.S3Config)
		if err != nil {
			return nil, err
		}
	case f.MinioConfig != nil:
		gf.Type = ControlPlaneFileStoreTypeMinio
		data, err = json.Marshal(f.MinioConfig)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid Config")
	}
	if err := json.Unmarshal(data, &gf.Config); err != nil {
		return nil, err
	}
	return json.Marshal(&gf)
}

// TBD
type FileStoreGCSConfig struct {
}

// TBD
type FileStoreS3Config struct {
}

type FileStoreMinioConfig struct {

	// +kubebuilder:validation:Format:=string

	// The address of Minio.
	Endpoint string `json:"endpoint"`

	// +kubebuilder:validation:Format:=string

	// The bucket name to store.
	Bucket string `json:"bucket"`

	// +kubebuilder:validation:Format:=string

	// The path to the access key file.
	AccessKeyFile string `json:"accessKeyFile"`

	// +kubebuilder:validation:Format:=string

	// The path to the secret key file.
	SecretKeyFile string `json:"secretKeyFile"`

	// +kubebuilder:validation:Format:=bool
	// +kubebuilder:validation:Optional

	// Whether the given bucket should be made automatically if not exists.
	AutoCreateBucket bool `json:"autoCreateBucket"`
}

type ControlPlaneCache struct {
	TTL int64 `json:"ttl"`
}

//type Duration time.Duration

type ControlPlaneProject struct {

	// +kubebuilder:validation:Format:=string
	// +kubebuilder:validation:Required

	// The unique identifier of the project.
	Id string `json:"id"`

	// +kubebuilder:validation:Format:=string
	// +kubebuilder:validation:Optional

	// The description about the project.
	Desc string `json:"desc"`

	// +kubebuilder:validation:Optional

	// Static admin account of the project.
	StaticAdmin ProjectStaticUser `json:"staticAdmin"`
}

type ProjectStaticUser struct {

	// +kubebuilder:validation:Format:=string
	// +kubebuilder:validation:Required

	// The username string.
	Username string `json:"username"`

	// +kubebuilder:validation:Format:=string
	// +kubebuilder:validation:Required

	// The bcrypt hashsed value of the password string.
	PasswordHash string `json:"passwordHash"`
}

type SharedSSOConfig struct {

	// +kubebuilder:validation:Format:=string

	Name   string          `json:"name"`
	GitHub SSOConfigGitHub `json:"github"`
	Google SSOConfigGoogle `json:"google"`
}

type SSOConfigGitHub struct {

	// +kubebuilder:validation:Format:=string

	ClientId string `json:"clientId"`

	// +kubebuilder:validation:Format:=string

	ClientSecret string `json:"clientSecret"`

	// +kubebuilder:validation:Format:=string

	BaseUrl string `json:"baseUrl"`

	// +kubebuilder:validation:Format:=string

	UploadUrl string `json:"uploadUrl"`
}

type SSOConfigGoogle struct {

	// +kubebuilder:validation:Format:=string

	ClientId string `json:"clientId"`

	// +kubebuilder:validation:Format:=string

	ClientSecret string `json:"clientSecret"`
}

// ControlPlaneStatus defines the observed state of ControlPlane
type ControlPlaneStatus struct {

	// +kubebuilder:validation:Format:=string

	// this is equal deployment.status.availableReplicas of gateway
	AvailableGatewayReplicas int32 `json:"availableGatewayReplicas"`

	// this is equal deployment.status.availableReplicas of api
	AvailableApiReplicas int32 `json:"availableApiReplicas"`

	// this is equal deployment.status.availableReplicas of web
	AvailableWebReplicas int32 `json:"availableWebReplicas"`

	// this is equal deployment.status.availableReplicas of cache
	AvailableCacheReplicas int32 `json:"availableCacheReplicas"`

	// this is equal deployment.status.availableReplicas of ops
	AvailableOpsReplicas int32 `json:"availableOpsReplicas"`

	// this is equal deployment.status.availableReplicas of minio
	AvailableMinioReplicas int32 `json:"availableMinioReplicas"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ControlPlane is the Schema for the controlplanes API
type ControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControlPlaneSpec   `json:"spec,omitempty"`
	Status ControlPlaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ControlPlaneList contains a list of ControlPlane
type ControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControlPlane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ControlPlane{}, &ControlPlaneList{})
}
