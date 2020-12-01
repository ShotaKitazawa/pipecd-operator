package controlplane

import (
	"io/ioutil"

	"github.com/ShotaKitazawa/pipecd-operator/embed"
)

func GenerateGatewayConfig() ([]byte, error) {
	f, err := embed.Assets.Open("/assets/envoy-config.yaml")
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}
