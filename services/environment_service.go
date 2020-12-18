package services

import (
	"context"
	"fmt"

	"github.com/ShotaKitazawa/pipecd-operator/infrastructure/pipecdweb"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/object"
	webservicepb "github.com/pipe-cd/pipe/pkg/app/api/service/webservice"
	"github.com/pipe-cd/pipe/pkg/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EnvironmentService struct {
	client.Client
	pipeCDWebClient webservicepb.WebServiceClient
}

func NewEnvironmentService(
	ctx context.Context,
	c client.Client,
	namespace string,
	projectId string,
	insecure bool,
) (*EnvironmentService, context.Context, error) {
	s, ctx, err := pipecdweb.NewPipeCDWebService(ctx, c, namespace, object.SecretName, object.SecretKeyEncryptionKey, projectId, insecure)
	if err != nil {
		return nil, ctx, err
	}
	return &EnvironmentService{c, s}, ctx, nil
}

func (s *EnvironmentService) Find(ctx context.Context, name string) (*model.Environment, error) {
	resp, err := s.pipeCDWebClient.ListEnvironments(ctx, &webservicepb.ListEnvironmentsRequest{})
	if err != nil {
		return nil, err
	}
	var envList []*model.Environment
	for _, respEnvironment := range resp.Environments {
		if name == respEnvironment.Name {
			envList = append(envList, respEnvironment)
		}
	}
	if len(envList) == 0 {
		return nil, fmt.Errorf(`no such Environment %v`, name)
	} else if len(envList) > 1 {
		// TODO: handle this error (something wrong)
		return nil, fmt.Errorf(`1 more than Environment %v (unknown error!!!!)`, name)
	}
	return envList[0], nil
}

func (s *EnvironmentService) Create(ctx context.Context, name, description string) (*model.Environment, error) {
	// add Environment
	if _, err := s.pipeCDWebClient.AddEnvironment(ctx, &webservicepb.AddEnvironmentRequest{Name: name, Desc: description}); err != nil {
		return nil, err
	}

	// get Environment (in the above)
	return s.Find(ctx, name)
}

func (s *EnvironmentService) CreateIfNotExists(ctx context.Context, name, description string) (*model.Environment, error) {
	env, err := s.Find(ctx, name)
	if err != nil {
		return s.Create(ctx, name, description)
	}
	return env, nil
}

func (s *EnvironmentService) Delete(ctx context.Context, id string) error {
	// TBD: PipeCD v0.9.0, no method DELETE of PipeCD Environment.
	return nil
}
