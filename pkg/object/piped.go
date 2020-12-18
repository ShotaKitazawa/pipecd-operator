package object

import (
	"fmt"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
)

const (
	pipedMinReplicas int32 = 1

	pipedContainerName          = "piped"
	pipedContainerImage         = "gcr.io/pipecd/piped"
	pipedContainerAdminPortName = "admin"
	pipedContainerAdminPort     = 9085
	pipedContainerHealthPath    = "/healthz"
	pipedContainerConfigName    = "piped-config"
	pipedContainerConfigPath    = "/etc/piped-config"
	pipedContainerSecretName    = "piped-secret"
	pipedContainerSecretPath    = "/etc/piped-secret"

	pipedServiceAdminPort     = 9085
	pipedServiceAdminPortName = "admin"

	ConfigMapPipedApiVersion = "pipecd.dev/v1beta1"
	ConfigMapPipedFilename   = "piped-config.yaml"

	pipedSecretKeyPipedKey = "piped-key"

	pipedClusterRoleBindingRoleRefAPIGroup = "rbac.authorization.k8s.io"
	pipedClusterRoleBindingRoleRefKind     = "ClusterRole"
	pipedClusterRoleBindingSubjectKind     = "ServiceAccount"
)

var (
	pipedContainerArgs = []string{
		"piped",
		"--config-file=/etc/piped-config/piped-config.yaml",
		"--metrics=true",
		"--use-fake-api-client=false",
		"--enable-default-kubernetes-cloud-provider=true",
		"--insecure=true",
		"--log-encoding=humanize",
	}
)

func MakePipedNamespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
}

func MakePipedDeploymentSpec(
	c pipecdv1alpha1.Piped,
	pipedId string,
) (appsv1.DeploymentSpec, error) {
	var replicas int32 = pipedMinReplicas

	podSpec, err := MakePipedPodSpec(c)
	if err != nil {
		return appsv1.DeploymentSpec{}, err
	}

	return appsv1.DeploymentSpec{
		Replicas: &replicas,
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RecreateDeploymentStrategyType,
		},
		Selector: &metav1.LabelSelector{
			MatchLabels: generatePipedLabel(),
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: generatePipedLabel(),
				Annotations: map[string]string{
					"piped-id": pipedId,
				},
			},
			Spec: podSpec,
		},
	}, nil
}

func MakePipedPodSpec(
	p pipecdv1alpha1.Piped,
) (corev1.PodSpec, error) {

	var (
		runAsNonRoot bool  = true
		runAsUser    int64 = 1000
	)
	spec := corev1.PodSpec{
		ServiceAccountName: p.Name,
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: &runAsNonRoot,
			RunAsUser:    &runAsUser,
			FSGroup:      &runAsUser,
		},
		Containers: []corev1.Container{
			{
				Name:            pipedContainerName,
				Image:           fmt.Sprintf("%s:%s", pipedContainerImage, p.Spec.Version),
				ImagePullPolicy: corev1.PullIfNotPresent,
				Args:            pipedContainerArgs,
				Ports: []corev1.ContainerPort{
					{
						Name:          pipedContainerAdminPortName,
						ContainerPort: pipedContainerAdminPort,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				LivenessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: pipedContainerHealthPath,
							Port: intstr.FromString(pipedContainerAdminPortName),
						},
					},
				},
				ReadinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: pipedContainerHealthPath,
							Port: intstr.FromString(pipedContainerAdminPortName),
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      pipedContainerConfigName,
						MountPath: pipedContainerConfigPath,
						ReadOnly:  true,
					},
					{
						Name:      pipedContainerSecretName,
						MountPath: pipedContainerSecretPath,
						ReadOnly:  true,
					},
				},
			},
		},
		Volumes: []corev1.Volume{
			{
				Name: pipedContainerConfigName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: p.Name,
						},
					},
				},
			},
			{
				Name: pipedContainerSecretName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: p.Name,
					},
				},
			},
		},
	}

	return spec, nil
}

func MakePipedServiceSpec(
	p pipecdv1alpha1.Piped,
) (corev1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Ports: []v1.ServicePort{
			{
				Name:       pipedServiceAdminPortName,
				Port:       pipedServiceAdminPort,
				TargetPort: intstr.FromString(pipedContainerAdminPortName),
			},
		},
		Selector: generatePipedLabel(),
	}, nil
}

func MakePipedSecretData(
	p pipecdv1alpha1.Piped,
	key string,
) map[string]string {
	result := make(map[string]string)
	result[pipedSecretKeyPipedKey] = key
	return result
}

func MakePipedConfigMapData(
	p pipecdv1alpha1.Piped,
	pipedId string,
) (map[string]string, error) {
	cm := make(map[string]string)
	config := pipecdv1alpha1.PipedConfig{
		Kind:       "Piped",
		APIVersion: ConfigMapPipedApiVersion,
		Spec:       p.Spec.Config,
	}
	config.Spec.ProjectID = p.Spec.ProjectId
	config.Spec.PipedKeyFile = filepath.Join(pipedContainerSecretPath, pipedSecretKeyPipedKey)
	config.Spec.PipedID = pipedId

	data, err := yaml.Marshal(&config)
	if err != nil {
		return nil, err
	}

	cm[ConfigMapPipedFilename] = string(data)
	return cm, nil
}

func MakePipedClusterRoleRules(
	p pipecdv1alpha1.Piped,
) []rbacv1.PolicyRule {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		},
		{
			NonResourceURLs: []string{"*"},
			Verbs:           []string{"*"},
		},
	}
}

func MakePipedClusterRoleBindingRoleRef(
	p pipecdv1alpha1.Piped,
) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: pipedClusterRoleBindingRoleRefAPIGroup,
		Kind:     pipedClusterRoleBindingRoleRefKind,
		Name:     p.Name,
	}
}

func MakePipedClusterRoleBindingSubjects(
	p pipecdv1alpha1.Piped,
) []rbacv1.Subject {
	return []rbacv1.Subject{
		{
			Kind:      pipedClusterRoleBindingSubjectKind,
			Name:      p.Name,
			Namespace: p.Namespace,
		},
	}
}

func generatePipedLabel() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     "piped",
		"app.kubernetes.io/instance": "piped",
	}
}
