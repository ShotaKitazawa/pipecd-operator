package pipecd

import (
	"context"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	webservicepb "github.com/pipe-cd/pipe/pkg/app/api/service/webservice"
	pipecd_jwt "github.com/pipe-cd/pipe/pkg/jwt"
	pipecd_model "github.com/pipe-cd/pipe/pkg/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ClientGenerator struct {
	encryptionKey string
	projectId     string
	insecure      bool
	certFile      string
}

func NewClientGenerator(encryptionKey, projectId string, insecure bool) ClientGenerator {
	return ClientGenerator{
		encryptionKey: encryptionKey,
		projectId:     projectId,
		insecure:      insecure,
	}
}

func (c ClientGenerator) WithCertFile(certFile string) ClientGenerator {
	c.certFile = certFile
	return c
}

func (c ClientGenerator) GeneratePipeCdWebServiceClient(ctx context.Context, addr string) (webservicepb.WebServiceClient, context.Context, error) {
	// generate signed token
	signedToken, err := c.generateSignedToken()
	if err != nil {
		return nil, ctx, err
	}

	// config context
	ctx = c.addSignedTokenToMetadata(ctx, signedToken)

	// create connection
	conn, err := grpc.DialContext(ctx, addr, c.parseOptions()...)
	if err != nil {
		return nil, ctx, err
	}
	return webservicepb.NewWebServiceClient(conn), ctx, nil
}

func (c ClientGenerator) generateSignedToken() (string, error) {
	now := time.Now()
	claims := &pipecd_jwt.Claims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "PipeCD",
			IssuedAt:  now.Unix(),
			NotBefore: now.Unix(),
			ExpiresAt: now.Add(time.Hour * 1).Unix(),
			Subject:   "TODO",
		},
		Role: pipecd_model.Role{
			ProjectId:   c.projectId,
			ProjectRole: pipecd_model.Role_ADMIN,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString([]byte(c.encryptionKey))
	if err != nil {
		return "", err
	}
	return t, nil
}

func (c ClientGenerator) addSignedTokenToMetadata(ctx context.Context, signedToken string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "cookie", fmt.Sprintf("token=%s", signedToken))
}

func (c ClientGenerator) parseOptions() []grpc.DialOption {
	var options []grpc.DialOption
	if !c.insecure {
		// TODO
	} else {
		options = append(options, grpc.WithInsecure())
	}
	return options
}
