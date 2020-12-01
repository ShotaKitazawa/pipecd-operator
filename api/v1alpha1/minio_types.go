package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MinioSpec defines the desired state of Minio
type MinioSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0

	// number of replicas
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// version of docker.io/minio/minio
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// secret for PipeCD EncryptionKey & Minio AccessKey and SecretKey
	Secret *v1.SecretVolumeSource `json:"secret"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// Storage spec to specify how storage shall be used.
	// Reference of https://github.com/prometheus-operator/prometheus-operator
	Storage *StorageSpec `json:"storage,omitempty"`
}

// MinioStatus defines the observed state of Minio
type MinioStatus struct {
	// this is equal deployment.status.availableReplicas of mongodb
	// +optional
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Minio is the Schema for the minios API
type Minio struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MinioSpec   `json:"spec,omitempty"`
	Status MinioStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MinioList contains a list of Minio
type MinioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Minio `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Minio{}, &MinioList{})
}
