package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// +kubebuilder:validation:Required

	Name string `json:"name"`

	// +kubebuilder:validation:Required

	ProjectId string `json:"projectID"`

	// +kubebuilder:validation:Required

	Kind string `json:"kind"`

	// +kubebuilder:validation:Required

	EnvironmentRef EnvironmentRef `json:"environmentRef"`

	// +kubebuilder:validation:Required

	PipedRef PipedRef `json:"pipedRef"`

	// +kubebuilder:validation:Required

	Repository string `json:"repository"`

	// +kubebuilder:validation:Required

	Path string `json:"path"`

	// +kubebuilder:validation:Optional

	ConfigFilename string `json:"configFilename"`

	// +kubebuilder:validation:Required

	CloudProvider string `json:"cloudProvider"`

	// +kubebuilder:validation:Required

	EncryptionKeyRef EmbeddedSecretKeyRef `json:"encryptionKeyRef"`

	// +kubebuilder:validation:Optional

	Insecure bool `json:"insecure"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
}

// +kubebuilder:object:root=true

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
