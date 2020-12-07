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

	ProjectID string `json:"projectID"`

	// +kubebuilder:validation:Optional

	Insecure bool `json:"insecure"`

	// +kubebuilder:validation:Optional

	EncryptionKeyRef EmbeddedSecretKeySelector `json:"encryptionKeyRef"`
}

type EmbeddedSecretKeySelector struct {
	// +kubebuilder:validation:Required

	// The name of the secret in the pod's namespace to select from.
	SecretName string `json:"secretName"`

	// +kubebuilder:validation:Required

	// The key of the secret to select from.  Must be a valid secret key.
	Key string `json:"key"`

	// +kubebuilder:validation:Optional

	// Specify whether the Secret or its key must be defined
	Optional *bool `json:"optional,omitempty"`
}

// EnvironmentStatus defines the observed state of Environment
type EnvironmentStatus struct {
	// TBD
}

// +kubebuilder:object:root=true

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
