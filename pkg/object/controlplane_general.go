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

	SecretName             = "pipecd"
	SecretKeyEncryptionKey = "encryption-key"

	ServiceName = "pipecd"
	servicePort = 8080
)

func MakeConfigMapBinaryData(
	c pipecdv1alpha1.ControlPlane,
) (map[string][]byte, error) {
	// deep copy
	copied := c.DeepCopy()

	cm := make(map[string][]byte)
	config := pipecdv1alpha1.ControlPlaneConfig{
		Kind:       "ControlPlane",
		APIVersion: ConfigMapControlPlaneApiVersion,
		Spec:       copied.Spec.Config,
	}
	data, err := json.Marshal(&config)
	if err != nil {
		return nil, err
	}

	cm[ConfigMapControlPlaneFilename] = data
	return cm, nil
}

func MakeSecretData(
	c pipecdv1alpha1.ControlPlane,
) map[string]string {
	result := make(map[string]string)
	//result[SecretKeyEncryptionKey] = utils.RandString(64)
	result[SecretKeyEncryptionKey] = "encryption-key-just-used-for-quickstart" // TODO
	return result
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
