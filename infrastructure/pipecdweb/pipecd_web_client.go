package pipecdweb

import (
	"context"
	"fmt"

	"github.com/ShotaKitazawa/pipecd-operator/pkg/object"
	webservicepb "github.com/pipe-cd/pipe/pkg/app/api/service/webservice"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pipecdv1alpha1 "github.com/ShotaKitazawa/pipecd-operator/api/v1alpha1"
)

func NewPipeCDWebService(
	ctx context.Context,
	c client.Client,
	namespace string,
	secretName string,
	secretKey string,
	projectId string,
	insecure bool,
) (webservicepb.WebServiceClient, context.Context, error) {

	/* Load Secret (key: encryption-key) */
	var secret v1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: secretName}, &secret); err != nil {
		return nil, ctx, err
	}
	encryptionKeyBytes, ok := secret.Data[secretKey]
	if !ok {
		return nil, ctx, fmt.Errorf(`no such key "%v" in Secret %v`, secretKey, secretName)
	}

	/* Load & Validate ControlPlaneList */
	var cpList pipecdv1alpha1.ControlPlaneList
	if err := c.List(ctx, &cpList); err != nil {
		return nil, ctx, err
	} else if len(cpList.Items) != 1 {
		return nil, ctx, fmt.Errorf("ControlPlane is none or more than 1: %v", cpList.Items)
	}
	cp := cpList.Items[0]
	serverNN := object.MakeServerNamespacedName(cp.Name, cp.Namespace)
	pipecdServerAddr := fmt.Sprintf("%s.%s.svc:%d",
		serverNN.Name,
		serverNN.Namespace,
		object.ServerContainerWebServerPort,
	)

	// Generate PipeCD WebService Client
	webServiceClient, ctx, err := NewPipeCDWebServiceClientGenerator(string(encryptionKeyBytes), projectId, insecure).
		GeneratePipeCdWebServiceClient(ctx, pipecdServerAddr)
	if err != nil {
		return nil, ctx, err
	}

	return webServiceClient, ctx, nil
}
