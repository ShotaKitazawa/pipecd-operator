package object

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
)

const (
	ConfigMapName                   = "pipecd"
	ConfigMapControlPlaneFilename   = "control-plane-config.yaml"
	ConfigMapControlPlaneApiVersion = "pipecd.dev/v1beta1"

	SecretKeyEncryptionKey  = "encryption-key"
	SecretKeyMinioAccessKey = "minio-access-key"
	SecretKeyMinioSecretKey = "minio-secret-key"

	ServiceName = "pipecd"
	servicePort = 8080
)

func MakeConfigMapBinaryData(
	c pipecdv1alpha1.ControlPlane,
) (map[string][]byte, error) {
	cm := make(map[string][]byte)
	config := pipecdv1alpha1.ControlPlaneConfig{
		Kind:       "ControlPlane",
		APIVersion: ConfigMapControlPlaneApiVersion,
		Spec:       c.Spec.Config,
	}
	data, err := json.Marshal(&config)
	if err != nil {
		return nil, err
	}

	cm[ConfigMapControlPlaneFilename] = data
	return cm, nil
}

func MakeServiceSpec(
	c pipecdv1alpha1.ControlPlane,
) (v1.ServiceSpec, error) {

	return v1.ServiceSpec{
		Type: v1.ServiceTypeNodePort,
		Ports: []v1.ServicePort{
			{
				Name:       "service",
				Port:       servicePort,
				TargetPort: intstr.FromString(gatewayContainerPortDPName),
			},
		},
		Selector: generateGatewayLabel(),
	}, nil
}
