package controlplane

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
)

const (
	cacheMinReplicas int32 = 1

	cacheContainerName            = "cache"
	cacheContainerImage           = "redis"
	cacheContainerImageTagDefault = "5.0.5-alpine3.9"
	cacheContainerPortName        = "redis"
	cacheContainerPort            = 6379

	cacheServicePort     = 6379
	cacheServicePortName = "service"
)

func MakeCacheNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-cache", name),
		Namespace: namespace,
	}
}

func MakeCacheDeploymentSpec(
	c pipecdv1alpha1.ControlPlane,
) (appsv1.DeploymentSpec, error) {
	var replicas int32
	if c.Spec.ReplicasCache == nil {
		replicas = cacheMinReplicas
	} else {
		if *c.Spec.ReplicasCache < 0 {
			replicas = 0
		} else {
			replicas = *c.Spec.ReplicasCache
		}
	}

	podSpec, err := MakeCachePodSpec(c)
	if err != nil {
		return appsv1.DeploymentSpec{}, err
	}

	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: generateCacheLabel(c.Name),
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: generateCacheLabel(c.Name),
			},
			Spec: podSpec,
		},
	}, nil
}

func MakeCachePodSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.PodSpec, error) {

	var image string
	if c.Spec.RedisVersion != "" {
		image = fmt.Sprintf("%s:%s", cacheContainerImage, c.Spec.RedisVersion)
	} else {
		image = fmt.Sprintf("%s:%s", cacheContainerImage, cacheContainerImageTagDefault)
	}

	return v1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            cacheContainerName,
				Image:           image,
				ImagePullPolicy: v1.PullIfNotPresent,
				Ports: []v1.ContainerPort{
					{
						Name:          cacheContainerPortName,
						ContainerPort: cacheContainerPort,
						Protocol:      v1.ProtocolTCP,
					},
				},
			},
		},
	}, nil
}

func MakeCacheServiceSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:       cacheServicePortName,
				Port:       cacheServicePort,
				TargetPort: intstr.FromString(cacheContainerPortName),
			},
		},
		Selector: generateCacheLabel(c.Name),
	}, nil
}

func generateCacheLabel(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "pipecd",
		"app.kubernetes.io/instance":  "pipecd",
		"app.kubernetes.io/component": "cache",
		"name":                        name,
	}
}
