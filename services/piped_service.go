package services

import (
	"context"
	"fmt"

	"github.com/ShotaKitazawa/pipecd-operator/entity"
	"github.com/ShotaKitazawa/pipecd-operator/infrastructure/pipecdweb"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/object"
	"github.com/golang/protobuf/ptypes/wrappers"
	webservicepb "github.com/pipe-cd/pipe/pkg/app/api/service/webservice"
	"github.com/pipe-cd/pipe/pkg/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PipedService struct {
	client.Client
	pipeCDWebClient webservicepb.WebServiceClient
}

func NewPipedService(
	ctx context.Context,
	c client.Client,
	namespace string,
	projectId string,
	insecure bool,
) (*PipedService, context.Context, error) {
	s, ctx, err := pipecdweb.NewPipeCDWebService(ctx, c, namespace, object.SecretName, object.SecretKeyEncryptionKey, projectId, insecure)
	if err != nil {
		return nil, ctx, err
	}
	return &PipedService{c, s}, ctx, nil
}

func (s *PipedService) FindById(ctx context.Context, id string) (*entity.Piped, error) {
	resp, err := s.pipeCDWebClient.GetPiped(ctx, &webservicepb.GetPipedRequest{PipedId: id})
	if err != nil {
		return nil, err
	} else if resp.Piped.Disabled {
		return nil, fmt.Errorf(`no such Piped`)
	}
	return &entity.Piped{
		Id:     resp.Piped.Id,
		Name:   resp.Piped.Name,
		Desc:   resp.Piped.Desc,
		EnvIds: resp.Piped.EnvIds,
	}, nil
}
func (s *PipedService) FindByName(ctx context.Context, name string) (*entity.Piped, error) {
	resp, err := s.pipeCDWebClient.ListPipeds(ctx, &webservicepb.ListPipedsRequest{Options: &webservicepb.ListPipedsRequest_Options{Enabled: &wrappers.BoolValue{Value: true}}})
	if err != nil {
		return nil, err
	}
	var pipedList []*model.Piped
	for _, respPiped := range resp.Pipeds {
		if respPiped.Name == name {
			pipedList = append(pipedList, respPiped)
		}
	}
	if len(pipedList) == 0 {
		return nil, fmt.Errorf(`no such Piped %v`, name)
	} else if len(pipedList) > 1 {
		// TODO: handle this error (something wrong)
		return nil, fmt.Errorf(`1 more than Environment %v (unknown error!!!!)`, name)
	}
	return &entity.Piped{
		Id:     pipedList[0].Id,
		Name:   pipedList[0].Name,
		Desc:   pipedList[0].Desc,
		EnvIds: pipedList[0].EnvIds,
	}, nil
}

func (s *PipedService) Create(ctx context.Context, name, description, environmentId string) (*entity.Piped, error) {
	// add Piped
	respRegisterPiped, err := s.pipeCDWebClient.RegisterPiped(ctx, &webservicepb.RegisterPipedRequest{
		Name:   name,
		Desc:   description,
		EnvIds: []string{environmentId},
	})
	if err != nil {
		return nil, err
	}

	// get Piped (in the above)
	piped, err := s.FindById(ctx, respRegisterPiped.Id)
	if err != nil {
		return nil, err
	}
	return &entity.Piped{
		Id:     piped.Id,
		Name:   piped.Name,
		Desc:   piped.Desc,
		EnvIds: piped.EnvIds,
		Key:    respRegisterPiped.Key,
	}, nil
}

func (s *PipedService) Delete(ctx context.Context, id string) error {
	if _, err := s.pipeCDWebClient.DisablePiped(ctx, &webservicepb.DisablePipedRequest{PipedId: id}); err != nil {
		return err
	}
	return nil
}
