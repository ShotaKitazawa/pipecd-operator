package controlplane

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
	"github.com/ShotaKitazawa/pipecd-operator/embed"
)

const (
	gatewayMinReplicas int32 = 1

	gatewayContainerName            = "gateway"
	gatewayContainerImage           = "envoyproxy/envoy-alpine"
	gatewayContainerImageTagDefault = "v1.10.0"
	gatewayContainerPortCPName      = "envoy-admin"
	gatewayContainerPortCPPort      = 9095
	gatewayContainerPortDPName      = "ingress"
	gatewayContainerPortDPPort      = 9090
	gatewayContainerHealthPath      = "/server_info"
	gatewayContainerConfigName      = "envoy-config"
	gatewayContainerConfigPath      = "/etc/envoy"
	gatewayContainerSecretName      = "pipecd-secret"
	gatewayContainerSecretPath      = "/etc/pipecd-secret"

	gatewayServicePort     = 9095
	gatewayServicePortName = "envoy-admin"
)

var (
	gatewayAssetFileNames = []string{
		"/assets/envoy-config.yaml",
	}
)

func MakeGatewayNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-gateway", name),
		Namespace: namespace,
	}
}

func MakeGatewayDeploymentSpec(
	c pipecdv1alpha1.ControlPlane,
	inputHash string,
) (appsv1.DeploymentSpec, error) {
	var replicas int32
	if c.Spec.ReplicasGateway == nil {
		replicas = gatewayMinReplicas
	} else {
		if *c.Spec.ReplicasGateway < 0 {
			replicas = 0
		} else {
			replicas = *c.Spec.ReplicasGateway
		}
	}

	podSpec, err := MakeGatewayPodSpec(c)
	if err != nil {
		return appsv1.DeploymentSpec{}, err
	}

	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: generateGatewayLabel(c.Name),
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: generateGatewayLabel(c.Name),
			},
			Spec: podSpec,
		},
	}, nil
}

func MakeGatewayPodSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.PodSpec, error) {

	nn := MakeGatewayNamespacedName(c.Name, c.Namespace)

	var image string
	if c.Spec.EnvoyVersion != "" {
		image = fmt.Sprintf("%s:%s", gatewayContainerImage, c.Spec.EnvoyVersion)
	} else {
		image = fmt.Sprintf("%s:%s", gatewayContainerImage, gatewayContainerImageTagDefault)
	}

	return v1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            gatewayContainerName,
				Image:           image,
				ImagePullPolicy: v1.PullIfNotPresent,
				Command:         []string{"envoy"},
				Args:            []string{"-c", "/etc/envoy/envoy-config.yaml"},
				Ports: []v1.ContainerPort{
					{
						Name:          gatewayContainerPortDPName,
						ContainerPort: gatewayContainerPortDPPort,
						Protocol:      v1.ProtocolTCP,
					},
					{
						Name:          gatewayContainerPortCPName,
						ContainerPort: gatewayContainerPortCPPort,
						Protocol:      v1.ProtocolTCP,
					},
				},
				LivenessProbe: &v1.Probe{
					InitialDelaySeconds: 15,
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: gatewayContainerHealthPath,
							Port: intstr.FromString(gatewayContainerPortCPName),
						},
					},
				},
				ReadinessProbe: &v1.Probe{
					InitialDelaySeconds: 15,
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: gatewayContainerHealthPath,
							Port: intstr.FromString(gatewayContainerPortCPName),
						},
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      gatewayContainerConfigName,
						MountPath: gatewayContainerConfigPath,
						ReadOnly:  true,
					},
					{
						Name:      gatewayContainerSecretName,
						MountPath: gatewayContainerSecretPath,
						ReadOnly:  true,
					},
				},
			},
		},
		Volumes: []v1.Volume{
			{
				Name: gatewayContainerConfigName,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: nn.Name,
						},
					},
				},
			},
			{
				Name: gatewayContainerSecretName,
				VolumeSource: v1.VolumeSource{
					Secret: &v1.SecretVolumeSource{
						SecretName: c.Spec.Secret.SecretName,
					},
				},
			},
		},
	}, nil
}

func MakeGatewayServiceSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:       gatewayServicePortName,
				Port:       gatewayServicePort,
				TargetPort: intstr.FromString(gatewayContainerPortCPName),
			},
		},
		Selector: generateGatewayLabel(c.Name),
	}, nil
}

func MakeGatewayConfigMapBinaryData(
	c pipecdv1alpha1.ControlPlane,
) (map[string][]byte, error) {

	data := make(map[string][]byte)

	for _, filename := range gatewayAssetFileNames {
		f, err := embed.Assets.Open(filename)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}

		bStr := string(b)
		bStr = strings.ReplaceAll(bStr, "ADDRESS_SERVER", MakeServerNamespacedName(c.Name, c.Namespace).Name)

		data[filepath.Base(filename)] = []byte(bStr)
	}

	return data, nil
}

func generateGatewayLabel(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "pipecd",
		"app.kubernetes.io/instance":  "pipecd",
		"app.kubernetes.io/component": "gateway",
		"name":                        name,
	}
}
