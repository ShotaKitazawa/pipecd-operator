package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnvironmentSpec defines the desired state of Environment
type EnvironmentSpec struct {
	// +kubebuilder:validation:Required

	Name string `json:"name"`

	// +kubebuilder:validation:Required

	Description string `json:"description"`

	// +kubebuilder:validation:Required

	ProjectId string `json:"projectID"`

	// +kubebuilder:validation:Required

	EncryptionKeyRef EmbeddedSecretKeyRef `json:"encryptionKeyRef"`

	// +kubebuilder:validation:Optional

	Insecure bool `json:"insecure"`
}

// EnvironmentStatus defines the observed state of Environment
type EnvironmentStatus struct {
	// +kubebuilder:validation:Optional

	EnvironmentId string `json:"environmentID"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Environment is the Schema for the environments API
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec,omitempty"`
	Status EnvironmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvironmentList contains a list of Environment
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Environment{}, &EnvironmentList{})
}
