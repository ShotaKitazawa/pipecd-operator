package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PipedSpec defines the desired state of Piped
type PipedSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Piped. Edit Piped_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// PipedStatus defines the observed state of Piped
type PipedStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

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
