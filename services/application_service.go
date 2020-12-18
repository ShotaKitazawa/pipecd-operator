package services

import (
	"context"
	"fmt"

	"github.com/ShotaKitazawa/pipecd-operator/infrastructure/pipecdweb"
	"github.com/ShotaKitazawa/pipecd-operator/pkg/object"
	"github.com/golang/protobuf/ptypes/wrappers"
	webservicepb "github.com/pipe-cd/pipe/pkg/app/api/service/webservice"
	"github.com/pipe-cd/pipe/pkg/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplicationService struct {
	client.Client
	pipeCDWebClient webservicepb.WebServiceClient
}

func NewApplicationService(
	ctx context.Context,
	c client.Client,
	namespace string,
	projectId string,
	insecure bool,
) (*ApplicationService, context.Context, error) {
	s, ctx, err := pipecdweb.NewPipeCDWebService(ctx, c, namespace, object.SecretName, object.SecretKeyEncryptionKey, projectId, insecure)
	if err != nil {
		return nil, ctx, err
	}
	return &ApplicationService{c, s}, ctx, nil
}

func (s *ApplicationService) FindByName(ctx context.Context, name string) (*model.Application, error) {
	resp, err := s.pipeCDWebClient.ListApplications(ctx, &webservicepb.ListApplicationsRequest{Options: &webservicepb.ListApplicationsRequest_Options{Enabled: &wrappers.BoolValue{Value: true}}})
	if err != nil {
		return nil, err
	}
	var envList []*model.Application
	for _, respApplication := range resp.Applications {
		if name == respApplication.Name {
			envList = append(envList, respApplication)
		}
	}
	if len(envList) == 0 {
		return nil, fmt.Errorf(`no such Application %v`, name)
	} else if len(envList) > 1 {
		// TODO: handle this error (something wrong)
		return nil, fmt.Errorf(`1 more than Application %v (unknown error!!!!)`, name)
	}
	return envList[0], nil
}

func (s *ApplicationService) Create(
	ctx context.Context,
	name,
	envId,
	pipedId,
	gitRepoId,
	gitBranch,
	gitConfigPath,
	gitConfigFileName,
	cloudProvider string,
	kind model.ApplicationKind,
) (*model.Application, error) {

	if gitConfigFileName == "" {
		gitConfigFileName = ".pipe.yaml"
	}

	// add Application
	if _, err := s.pipeCDWebClient.AddApplication(ctx, &webservicepb.AddApplicationRequest{
		Name:    name,
		EnvId:   envId,
		PipedId: pipedId,
		GitPath: &model.ApplicationGitPath{
			Repo: &model.ApplicationGitRepository{
				Id: gitRepoId,
				// Remote: TODO,
				Branch: gitBranch,
			},
			ConfigPath:     gitConfigPath,
			ConfigFilename: gitConfigFileName,
		},
		Kind:          kind,
		CloudProvider: cloudProvider,
	}); err != nil {
		return nil, err
	}

	// get Application (in the above)
	return s.FindByName(ctx, name)
}

func (s *ApplicationService) CreateIfNotExists(
	ctx context.Context,
	name,
	envId,
	pipedId,
	gitRepoId,
	gitBranch,
	gitConfigPath,
	gitConfigFileName,
	cloudProvider string,
	kind model.ApplicationKind,

) (*model.Application, error) {
	env, err := s.FindByName(ctx, name)
	if err != nil {
		return s.Create(ctx, name, envId, pipedId, gitRepoId, gitBranch, gitConfigPath, gitConfigFileName, cloudProvider, kind)
	}
	return env, nil
}

func (s *ApplicationService) Delete(ctx context.Context, id string) error {
	if _, err := s.pipeCDWebClient.DisableApplication(ctx, &webservicepb.DisableApplicationRequest{ApplicationId: id}); err != nil {
		return err
	}
	return nil
}
