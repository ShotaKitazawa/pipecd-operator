package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MongoSpec defines the desired state of Mongo
type MongoSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0

	// number of replicas
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// version of docker.io/mongo
	Version string `json:"version"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Format:=string

	// Storage spec to specify how storage shall be used.
	// Reference of https://github.com/prometheus-operator/prometheus-operator
	Storage *StorageSpec `json:"storage,omitempty"`
}

// MongoStatus defines the observed state of Mongo
type MongoStatus struct {
	// this is equal deployment.status.availableReplicas of mongodb
	AvailableReplicas int32 `json:"availableReplicas"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Mongo is the Schema for the mongoes API
type Mongo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MongoSpec   `json:"spec,omitempty"`
	Status MongoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MongoList contains a list of Mongo
type MongoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mongo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Mongo{}, &MongoList{})
}
