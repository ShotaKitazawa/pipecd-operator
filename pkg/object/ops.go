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
	opsMinReplicas int32 = 1

	opsContainerName          = "ops"
	opsContainerImage         = "gcr.io/pipecd/ops"
	opsContainerOpsPortName   = "ops"
	opsContainerOpsPort       = 9082
	opsContainerAdminPortName = "admin"
	opsContainerAdminPort     = 9085
	opsContainerHealthPath    = "/healthz"
	opsContainerConfigName    = "pipecd-config"
	opsContainerConfigPath    = "/etc/pipecd-config"

	opsServiceOpsPort       = 9082
	opsServiceOpsPortName   = "piped-ops"
	opsServiceAdminPort     = 9085
	opsServiceAdminPortName = "admin"
)

var (
	opsContainerArgs = []string{
		"server",
		"--config-file=/etc/pipecd-config/control-plane-config.yaml",
	}
)

func MakeOpsNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      fmt.Sprintf("%s-ops", name),
		Namespace: namespace,
	}
}

func MakeOpsDeploymentSpec(
	c pipecdv1alpha1.ControlPlane,
) (appsv1.DeploymentSpec, error) {
	var replicas int32
	if c.Spec.ReplicasOps == nil {
		replicas = opsMinReplicas
	} else {
		if *c.Spec.ReplicasOps < 0 {
			replicas = 0
		} else {
			replicas = *c.Spec.ReplicasOps
		}
	}

	podSpec, err := MakeOpsPodSpec(c)
	if err != nil {
		return appsv1.DeploymentSpec{}, err
	}

	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: generateOpsLabel(),
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: generateOpsLabel(),
			},
			Spec: podSpec,
		},
	}, nil
}

func MakeOpsPodSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.PodSpec, error) {

	spec := v1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            opsContainerName,
				Image:           fmt.Sprintf("%s:%s", opsContainerImage, c.Spec.Version),
				ImagePullPolicy: v1.PullIfNotPresent,
				Args:            opsContainerArgs,
				Ports: []v1.ContainerPort{
					{
						Name:          opsContainerOpsPortName,
						ContainerPort: opsContainerOpsPort,
						Protocol:      v1.ProtocolTCP,
					},
					{
						Name:          opsContainerAdminPortName,
						ContainerPort: opsContainerAdminPort,
						Protocol:      v1.ProtocolTCP,
					},
				},
				LivenessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: opsContainerHealthPath,
							Port: intstr.FromString(opsContainerAdminPortName),
						},
					},
				},
				ReadinessProbe: &v1.Probe{
					Handler: v1.Handler{
						HTTPGet: &v1.HTTPGetAction{
							Path: opsContainerHealthPath,
							Port: intstr.FromString(opsContainerAdminPortName),
						},
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      opsContainerConfigName,
						MountPath: opsContainerConfigPath,
						ReadOnly:  true,
					},
				},
			},
		},
		Volumes: []v1.Volume{
			{
				Name: opsContainerConfigName,
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						LocalObjectReference: v1.LocalObjectReference{
							Name: ConfigMapName,
						},
					},
				},
			},
		},
	}

	for _, v := range c.Spec.VolumeMounts {
		spec.Volumes = append(spec.Volumes, v.Volume)
		for idx := 0; idx < len(spec.Containers); idx++ {
			spec.Containers[idx].VolumeMounts = append(spec.Containers[idx].VolumeMounts, corev1.VolumeMount{
				Name:        v.Volume.Name,
				MountPath:   v.MountPath,
				ReadOnly:    v.ReadOnly,
				SubPath:     v.SubPath,
				SubPathExpr: v.SubPathExpr,
			})
		}
	}
	return spec, nil
}

func MakeOpsServiceSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:       opsServiceOpsPortName,
				Port:       opsServiceOpsPort,
				TargetPort: intstr.FromString(opsContainerOpsPortName),
			},
			{
				Name:       opsServiceAdminPortName,
				Port:       opsServiceAdminPort,
				TargetPort: intstr.FromString(opsContainerAdminPortName),
			},
		},
		Selector: generateOpsLabel(),
	}, nil
}

func generateOpsLabel() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "pipecd",
		"app.kubernetes.io/instance":  "pipecd",
		"app.kubernetes.io/component": "ops",
	}
}
