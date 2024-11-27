package main

import (
	"context"
	"fmt"
	"os"

	"github.com/conductorone/baton-freshservice/pkg/client"
	"github.com/conductorone/baton-freshservice/pkg/connector"
	configSchema "github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	version       = "dev"
	connectorName = "baton-freshservice"
	apiKey        = "api-key"
	domain        = "domain"
)

var (
	apiKeyField         = field.StringField(apiKey, field.WithRequired(true), field.WithDescription("The api key for your account."))
	domainField         = field.StringField(domain, field.WithRequired(true), field.WithDescription("The domain for your account."))
	configurationFields = []field.SchemaField{apiKeyField, domainField}
)

func main() {
	ctx := context.Background()
	_, cmd, err := configSchema.DefineConfiguration(ctx,
		connectorName,
		getConnector,
		field.NewConfiguration(configurationFields),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version
	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConnector(ctx context.Context, cfg *viper.Viper) (types.ConnectorServer, error) {
	var (
		fsClient = client.NewClient()
		fsToken  = cfg.GetString(apiKey)
		fsDomain = cfg.GetString(domain)
	)
	l := ctxzap.Extract(ctx)
	if fsToken != "" && fsDomain != "" {
		fsClient.WithBearerToken(fsToken).WithDomain(fsDomain)
	}

	cb, err := connector.New(ctx,
		fsToken,
		fsDomain,
		fsClient,
	)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	c, err := connectorbuilder.NewConnector(ctx, cb)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return c, nil
}
