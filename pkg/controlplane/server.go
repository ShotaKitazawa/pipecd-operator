package controlplane

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
	serverMinReplicas int32 = 1

	serverContainerName                = "server"
	serverContainerImage               = "gcr.io/pipecd/pipecd"
	serverContainerPipedServerPortName = "piped-api"
	serverContainerPipedServerPort     = 9080
	serverContainerWebServerPortName   = "web-api"
	serverContainerWebServerPort       = 9081
	serverContainerHttpPortName        = "http"
	serverContainerHttpPort            = 9082
	serverContainerAdminPortName       = "admin"
	serverContainerAdminPort           = 9085
	serverContainerHealthPath          = "/healthz"
	serverContainerConfigName          = "pipecd-config"
	serverContainerConfigPath          = "/etc/pipecd-config"
	serverContainerSecretName          = "pipecd-secret"
	serverContainerSecretPath          = "/etc/pipecd-secret"

	serverServicePipedServerPort     = 9080
	serverServicePipedServerPortName = "piped-server"
	serverServiceWebServerPort       = 9081
	serverServiceWebServerPortName   = "web-server"
	serverServiceHttpPort            = 9082
	serverServiceHttpPortName        = "http"
	serverServiceAdminPort           = 9085
	serverServiceAdminPortName       = "admin"
)

var (
	serverContainerArgs = []string{
		"server",
		"--insecure-cookie=true",
		"--config-file=/etc/pipecd-config/control-plane-config.yaml",
		"--enable-grpc-reflection=false",
		"--encryption-key-file=/etc/pipecd-secret/encryption-key",
		"--log-encoding=humanize",
	}
)

func MakeServerNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-server", name),
		Namespace: namespace,
	}
}

func MakeServerDeploymentSpec(
	c pipecdv1alpha1.ControlPlane,
) (appsv1.DeploymentSpec, error) {
	var replicas int32
	if c.Spec.ReplicasServer == nil {
		replicas = serverMinReplicas
	} else {
		if *c.Spec.ReplicasServer < 0 {
			replicas = 0
		} else {
			replicas = *c.Spec.ReplicasServer
		}
	}

	podSpec, err := MakeServerPodSpec(c)
	if err != nil {
		return appsv1.DeploymentSpec{}, err
	}

	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: generateServerLabel(c.Name),
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: generateServerLabel(c.Name),
			},
			Spec: podSpec,
		},
	}, nil
}

func MakeServerPodSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.PodSpec, error) {

	args := append(serverContainerArgs,
		fmt.Sprintf("--cache-address=%s:%d", MakeCacheNamespacedName(c.Name, c.Namespace).Name, cacheServicePort),
	)

	return v1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            serverContainerName,
				Image:           fmt.Sprintf("%s:%s", serverContainerImage, c.Spec.Version),
				ImagePullPolicy: v1.PullIfNotPresent,
				Args:            args,
				Ports: []v1.ContainerPort{
					{
						Name:          serverContainerPipedServerPortName,
						ContainerPort: serverContainerPipedServerPort,
						Protocol:      v1.ProtocolTCP,
					},
					{
						Name:          serverContainerWebServerPortName,
						ContainerPort: serverContainerWebServerPort,
						Protocol:      v1.ProtocolTCP,
					},
					{
						Name:          serverContainerHttpPortName,
						ContainerPort: serverContainerHttpPort,
						Protocol:      v1.ProtocolTCP,
					},
					{
						Name:          serverContainerAdminPortName,
						ContainerPort: serverContainerAdminPort,
						Protocol:      v1.ProtocolTCP,
					},
				},
				LivenessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: serverContainerHealthPath,
							Port: intstr.FromString(serverContainerAdminPortName),
						},
					},
				},
				ReadinessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: serverContainerHealthPath,
							Port: intstr.FromString(serverContainerAdminPortName),
						},
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      serverContainerConfigName,
						MountPath: serverContainerConfigPath,
						ReadOnly:  true,
					},
					{
						Name:      serverContainerSecretName,
						MountPath: serverContainerSecretPath,
						ReadOnly:  true,
					},
				},
			},
		},
		Volumes: []v1.Volume{
			{
				Name: serverContainerConfigName,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: ConfigMapName,
						},
					},
				},
			},
			{
				Name: serverContainerSecretName,
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: c.Spec.Secret.SecretName,
					},
				},
			},
		},
	}, nil
}

func MakeServerServiceSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:       serverServicePipedServerPortName,
				Port:       serverServicePipedServerPort,
				TargetPort: intstr.FromString(serverContainerPipedServerPortName),
			},
			{
				Name:       serverServiceWebServerPortName,
				Port:       serverServiceWebServerPort,
				TargetPort: intstr.FromString(serverContainerWebServerPortName),
			},
			{
				Name:       serverServiceHttpPortName,
				Port:       serverServiceHttpPort,
				TargetPort: intstr.FromString(serverContainerHttpPortName),
			},
			{
				Name:       serverServiceAdminPortName,
				Port:       serverServiceAdminPort,
				TargetPort: intstr.FromString(serverContainerAdminPortName),
			},
		},
		Selector: generateServerLabel(c.Name),
	}, nil
}

func generateServerLabel(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "pipecd",
		"app.kubernetes.io/instance":  "pipecd",
		"app.kubernetes.io/component": "server",
		"name":                        name,
	}
}
