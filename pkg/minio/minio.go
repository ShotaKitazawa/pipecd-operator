package minio

import (
	"fmt"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/controlplane"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	minioReplicas int32 = 1

	minioContainerName            = "minio"
	minioContainerImage           = "minio/minio"
	minioContainerImageTagDefault = "RELEASE.2020-08-26T00-00-49Z"
	minioContainerPortName        = "minio"
	minioContainerPort            = 9000

	minioVolumeName = "volume"

	minioServicePort     = 9000
	minioServicePortName = "service"
)

var (
	minioContainerArgs = []string{
		"server",
		"/data",
	}
)

func MakeMinioNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-minio", name),
		Namespace: namespace,
	}
}

func MakeMinioStatefulSetSpec(
	m pipecdv1alpha1.Minio,
) (appsv1.StatefulSetSpec, error) {
	var replicas int32 = minioReplicas

	podSpec, err := MakeMinioPodSpec(m)
	if err != nil {
		return appsv1.StatefulSetSpec{}, err
	}

	spec := appsv1.StatefulSetSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: generateMinioLabel(m.Name),
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: generateMinioLabel(m.Name),
			},
			Spec: podSpec,
		},
	}

	// TODO: this logic is "or", but should be "xor"
	switch {
	case m.Spec.Storage.EmptyDir != nil:
		// pass
	case m.Spec.Storage.VolumeClaimTemplate != nil:
		spec.VolumeClaimTemplates = []v1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: minioVolumeName,
				},
				Spec: m.Spec.Storage.VolumeClaimTemplate.Spec,
			},
		}
	}

	return spec, nil
}

func MakeMinioPodSpec(
	m pipecdv1alpha1.Minio,
) (v1.PodSpec, error) {

	var image string
	if m.Spec.Version != "" {
		image = fmt.Sprintf("%s:%s", minioContainerImage, m.Spec.Version)
	} else {
		image = fmt.Sprintf("%s:%s", minioContainerImage, minioContainerImageTagDefault)
	}

	spec := v1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            minioContainerName,
				Image:           image,
				ImagePullPolicy: v1.PullIfNotPresent,
				Args:            minioContainerArgs,
				Env: []v1.EnvVar{
					{
						Name: "MINIO_ACCESS_KEY",
						ValueFrom: &v1.EnvVarSource{
							SecretKeyRef: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: m.Spec.Secret.SecretName,
								},
								Key: controlplane.SecretKeyMinioAccessKey,
							},
						},
					},
					{
						Name: "MINIO_SECRET_KEY",
						ValueFrom: &v1.EnvVarSource{
							SecretKeyRef: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: m.Spec.Secret.SecretName,
								},
								Key: controlplane.SecretKeyMinioSecretKey,
							},
						},
					},
				},
				Ports: []v1.ContainerPort{
					{
						Name:          minioContainerPortName,
						ContainerPort: minioContainerPort,
						Protocol:      v1.ProtocolTCP,
					},
				},
			},
		},
	}

	// TODO: this logic is "or", but should be "xor"
	switch {
	case m.Spec.Storage.EmptyDir != nil:
		spec.Volumes = append(spec.Volumes, v1.Volume{
			Name: minioVolumeName,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
	case m.Spec.Storage.VolumeClaimTemplate != nil:
		// pass
	}

	return spec, nil
}

func MakeMinioServiceSpec(
	m pipecdv1alpha1.Minio,
) (v1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:       minioServicePortName,
				Port:       minioServicePort,
				TargetPort: intstr.FromString(minioContainerPortName),
			},
		},
		Selector: generateMinioLabel(m.Name),
	}, nil
}

func generateMinioLabel(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "pipecd",
		"app.kubernetes.io/instance":  "pipecd",
		"app.kubernetes.io/component": "minio",
		"name":                        name,
	}
}
