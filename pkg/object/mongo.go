package object

import (
	"fmt"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	mongoReplicas int32 = 1

	mongoContainerName            = "mongo"
	mongoContainerImage           = "mongo"
	mongoContainerImageTagDefault = "4.2"
	mongoContainerPortName        = "mongo"
	mongoContainerPort            = 27017

	mongoVolumeName = "volume"

	mongoServicePort     = 27017
	mongoServicePortName = "service"
)

func MakeMongoNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-mongo", name),
		Namespace: namespace,
	}
}

func MakeMongoStatefulSetSpec(
	m pipecdv1alpha1.Mongo,
) (appsv1.StatefulSetSpec, error) {
	var replicas int32 = mongoReplicas

	podSpec, err := MakeMongoPodSpec(m)
	if err != nil {
		return appsv1.StatefulSetSpec{}, err
	}

	spec := appsv1.StatefulSetSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: generateMongoLabel(m.Name),
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: generateMongoLabel(m.Name),
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
					Name: mongoVolumeName,
				},
				Spec: m.Spec.Storage.VolumeClaimTemplate.Spec,
			},
		}
	}

	return spec, nil
}

func MakeMongoPodSpec(
	m pipecdv1alpha1.Mongo,
) (v1.PodSpec, error) {

	var image string
	if m.Spec.Version != "" {
		image = fmt.Sprintf("%s:%s", mongoContainerImage, m.Spec.Version)
	} else {
		image = fmt.Sprintf("%s:%s", mongoContainerImage, mongoContainerImageTagDefault)
	}

	spec := v1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            mongoContainerName,
				Image:           image,
				ImagePullPolicy: v1.PullIfNotPresent,
				Ports: []v1.ContainerPort{
					{
						Name:          mongoContainerPortName,
						ContainerPort: mongoContainerPort,
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
			Name: mongoVolumeName,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		})
	case m.Spec.Storage.VolumeClaimTemplate != nil:
		// pass
	}

	return spec, nil
}

func MakeMongoServiceSpec(
	m pipecdv1alpha1.Mongo,
) (v1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:       mongoServicePortName,
				Port:       mongoServicePort,
				TargetPort: intstr.FromString(mongoContainerPortName),
			},
		},
		Selector: generateMongoLabel(m.Name),
	}, nil
}

func generateMongoLabel(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "pipecd",
		"app.kubernetes.io/instance":  "pipecd",
		"app.kubernetes.io/component": "mongo",
		"name":                        name,
	}
}
