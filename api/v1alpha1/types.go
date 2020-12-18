package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StorageSpec defines the configured storage for a group Mogno servers.
// If neither `emptyDir` nor `volumeClaimTemplate` is specified, then by default an [EmptyDir](https://kubernetes.io/docs/concepts/storage/volumes/#emptydir) will be used.
type StorageSpec struct {
	// EmptyDirVolumeSource to be used by the Prometheus StatefulSets. If specified, used in place of any volumeClaimTemplate. More
	// info: https://kubernetes.io/docs/concepts/storage/volumes/#emptydir
	EmptyDir *v1.EmptyDirVolumeSource `json:"emptyDir,omitempty"`

	// A PVC spec to be used by the Prometheus StatefulSets.
	VolumeClaimTemplate *EmbeddedPersistentVolumeClaim `json:"volumeClaimTemplate,omitempty"`
}

// EmbeddedPersistentVolumeClaim is an embedded version of k8s.io/api/core/v1.PersistentVolumeClaim.
// It contains TypeMeta and a reduced ObjectMeta.
type EmbeddedPersistentVolumeClaim struct {
	metav1.TypeMeta `json:",inline"`

	// Spec defines the desired characteristics of a volume requested by a pod author.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	Spec v1.PersistentVolumeClaimSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type EmbeddedVolumeMount struct {
	// +kubebuilder:validation:Required

	MountPath string `json:"mountPath"`

	// +kubebuilder:validation:Required

	Volume v1.Volume `json:"volume"`

	// +kubebuilder:validation:Optional

	ReadOnly bool `json:"readOnly"`

	// +kubebuilder:validation:Optional

	SubPath string `json:"subPath"`

	// +kubebuilder:validation:Optional

	SubPathExpr string `json:"subPathExpr"`
}

type EnvironmentRef struct {
	ObjectName string `json:"objectName"`
	Name       string `json:"name"`
}

type PipedRef struct {
	ObjectName string `json:"objectName"`
	Name       string `json:"name"`
}
