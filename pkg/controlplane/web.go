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
	webMinReplicas int32 = 1

	webContainerName          = "web"
	webContainerImage         = "gcr.io/pipecd/web"
	webContainerWebPortName   = "web"
	webContainerWebPort       = 9082
	webContainerAdminPortName = "admin"
	webContainerAdminPort     = 9085
	webContainerHealthPath    = "/healthz"

	webServiceWebPort       = 9082
	webServiceWebPortName   = "piped-web"
	webServiceAdminPort     = 9085
	webServiceAdminPortName = "admin"
)

var (
	webContainerArgs = []string{
		"server",
	}
)

func MakeWebNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-web", name),
		Namespace: namespace,
	}
}

func MakeWebDeploymentSpec(
	c pipecdv1alpha1.ControlPlane,
	inputHash string,
) (appsv1.DeploymentSpec, error) {
	var replicas int32
	if c.Spec.ReplicasWeb == nil {
		replicas = webMinReplicas
	} else {
		if *c.Spec.ReplicasWeb < 0 {
			replicas = 0
		} else {
			replicas = *c.Spec.ReplicasWeb
		}
	}

	podSpec, err := MakeWebPodSpec(c)
	if err != nil {
		return appsv1.DeploymentSpec{}, err
	}

	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: generateWebLabel(c.Name),
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: generateWebLabel(c.Name),
			},
			Spec: podSpec,
		},
	}, nil
}

func MakeWebPodSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.PodSpec, error) {

	return v1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            webContainerName,
				Image:           fmt.Sprintf("%s:%s", webContainerImage, c.Spec.Version),
				ImagePullPolicy: v1.PullIfNotPresent,
				Args:            webContainerArgs,
				Ports: []v1.ContainerPort{
					{
						Name:          webContainerWebPortName,
						ContainerPort: webContainerWebPort,
						Protocol:      v1.ProtocolTCP,
					},
					{
						Name:          webContainerAdminPortName,
						ContainerPort: webContainerAdminPort,
						Protocol:      v1.ProtocolTCP,
					},
				},
				LivenessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: webContainerHealthPath,
							Port: intstr.FromString(webContainerAdminPortName),
						},
					},
				},
				ReadinessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: webContainerHealthPath,
							Port: intstr.FromString(webContainerAdminPortName),
						},
					},
				},
			},
		},
	}, nil
}

func MakeWebServiceSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:       webServiceWebPortName,
				Port:       webServiceWebPort,
				TargetPort: intstr.FromString(webContainerWebPortName),
			},
			{
				Name:       webServiceAdminPortName,
				Port:       webServiceAdminPort,
				TargetPort: intstr.FromString(webContainerAdminPortName),
			},
		},
		Selector: generateWebLabel(c.Name),
	}, nil
}

func generateWebLabel(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "pipecd",
		"app.kubernetes.io/instance":  "pipecd",
		"app.kubernetes.io/component": "web",
		"name":                        name,
	}
}
