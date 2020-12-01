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
	apiMinReplicas int32 = 1

	apiContainerName             = "api"
	apiContainerImage            = "gcr.io/pipecd/api"
	apiContainerPipedApiPortName = "piped-api"
	apiContainerPipedApiPort     = 9080
	apiContainerWebApiPortName   = "web-api"
	apiContainerWebApiPort       = 9081
	apiContainerHttpPortName     = "http"
	apiContainerHttpPort         = 9082
	apiContainerAdminPortName    = "admin"
	apiContainerAdminPort        = 9085
	apiContainerHealthPath       = "/healthz"
	apiContainerConfigName       = "pipecd-config"
	apiContainerConfigPath       = "/etc/pipecd-config"
	apiContainerSecretName       = "pipecd-secret"
	apiContainerSecretPath       = "/etc/pipecd-secret"

	apiServicePipedApiPort     = 9080
	apiServicePipedApiPortName = "piped-api"
	apiServiceWebApiPort       = 9081
	apiServiceWebApiPortName   = "web-api"
	apiServiceHttpPort         = 9082
	apiServiceHttpPortName     = "http"
	apiServiceAdminPort        = 9085
	apiServiceAdminPortName    = "admin"
)

var (
	apiContainerArgs = []string{
		"server",
		"--insecure-cookie=true",
		"--config-file=/etc/pipecd-config/control-plane-config.yaml",
		"--use-fake-response=false",
		"--enable-grpc-reflection=true",
		"--encryption-key-file=/etc/pipecd-secret/encryption-key",
	}
)

func MakeApiNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-api", name),
		Namespace: namespace,
	}
}

func MakeApiDeploymentSpec(
	c pipecdv1alpha1.ControlPlane,
	inputHash string,
) (appsv1.DeploymentSpec, error) {
	var replicas int32
	if c.Spec.ReplicasApi == nil {
		replicas = apiMinReplicas
	} else {
		if *c.Spec.ReplicasApi < 0 {
			replicas = 0
		} else {
			replicas = *c.Spec.ReplicasApi
		}
	}

	podSpec, err := MakeApiPodSpec(c)
	if err != nil {
		return appsv1.DeploymentSpec{}, err
	}

	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: generateApiLabel(c.Name),
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: generateApiLabel(c.Name),
			},
			Spec: podSpec,
		},
	}, nil
}

func MakeApiPodSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.PodSpec, error) {

	args := append(apiContainerArgs,
		fmt.Sprintf("--cache-address=%s:%d", MakeCacheNamespacedName(c.Name, c.Namespace).Name, cacheServicePort),
	)

	return v1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            apiContainerName,
				Image:           fmt.Sprintf("%s:%s", apiContainerImage, c.Spec.Version),
				ImagePullPolicy: v1.PullIfNotPresent,
				Args:            args,
				Ports: []v1.ContainerPort{
					{
						Name:          apiContainerPipedApiPortName,
						ContainerPort: apiContainerPipedApiPort,
						Protocol:      v1.ProtocolTCP,
					},
					{
						Name:          apiContainerWebApiPortName,
						ContainerPort: apiContainerWebApiPort,
						Protocol:      v1.ProtocolTCP,
					},
					{
						Name:          apiContainerHttpPortName,
						ContainerPort: apiContainerHttpPort,
						Protocol:      v1.ProtocolTCP,
					},
					{
						Name:          apiContainerAdminPortName,
						ContainerPort: apiContainerAdminPort,
						Protocol:      v1.ProtocolTCP,
					},
				},
				LivenessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: apiContainerHealthPath,
							Port: intstr.FromString(apiContainerAdminPortName),
						},
					},
				},
				ReadinessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: apiContainerHealthPath,
							Port: intstr.FromString(apiContainerAdminPortName),
						},
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      apiContainerConfigName,
						MountPath: apiContainerConfigPath,
						ReadOnly:  true,
					},
					{
						Name:      apiContainerSecretName,
						MountPath: apiContainerSecretPath,
						ReadOnly:  true,
					},
				},
			},
		},
		Volumes: []v1.Volume{
			{
				Name: apiContainerConfigName,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: ConfigMapName,
						},
					},
				},
			},
			{
				Name: apiContainerSecretName,
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: c.Spec.Secret.SecretName,
					},
				},
			},
		},
	}, nil
}

func MakeApiServiceSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:       apiServicePipedApiPortName,
				Port:       apiServicePipedApiPort,
				TargetPort: intstr.FromString(apiContainerPipedApiPortName),
			},
			{
				Name:       apiServiceWebApiPortName,
				Port:       apiServiceWebApiPort,
				TargetPort: intstr.FromString(apiContainerWebApiPortName),
			},
			{
				Name:       apiServiceHttpPortName,
				Port:       apiServiceHttpPort,
				TargetPort: intstr.FromString(apiContainerHttpPortName),
			},
			{
				Name:       apiServiceAdminPortName,
				Port:       apiServiceAdminPort,
				TargetPort: intstr.FromString(apiContainerAdminPortName),
			},
		},
		Selector: generateApiLabel(c.Name),
	}, nil
}

func generateApiLabel(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "pipecd",
		"app.kubernetes.io/instance":  "pipecd",
		"app.kubernetes.io/component": "api",
		"name":                        name,
	}
}
