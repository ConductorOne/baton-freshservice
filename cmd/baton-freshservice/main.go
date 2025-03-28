package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

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
)

func main() {
	ctx := context.Background()
	_, cmd, err := configSchema.DefineConfiguration(ctx,
		connectorName,
		getConnector,
		field.NewConfiguration(configurationFields, configRelations...),
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

// extractSubdomain parses a URL or domain string and returns the subdomain portion.
func extractSubdomain(input string) (string, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		parsedURL, err := url.Parse(input)
		if err != nil {
			return "", fmt.Errorf("invalid domain URL: %w", err)
		}
		input = parsedURL.Hostname()
	}
	parts := strings.Split(input, ".")
	if len(parts) > 0 {
		return parts[0], nil
	}
	return "", nil
}

func getConnector(ctx context.Context, cfg *viper.Viper) (types.ConnectorServer, error) {
	fsClient := client.NewClient()
	fsToken := cfg.GetString(apiKey)
	fsDomain := cfg.GetString(domain)

	l := ctxzap.Extract(ctx)
	subdomain, err := extractSubdomain(fsDomain)
	if err != nil {
		l.Error("error extracting subdomain", zap.Error(err))
		return nil, err
	}
	fsDomain = subdomain

	// Validate the subdomain format
	if strings.Contains(fsDomain, ".") || strings.Contains(fsDomain, "/") || fsDomain == "" {
		return nil, fmt.Errorf("invalid subdomain format: %q - should be just the subdomain portion (e.g., 'company' not 'company.freshservice.com')", fsDomain)
	}

	categoryId := cfg.GetString(categoryField.FieldName)

	fsClient = fsClient.WithBearerToken(fsToken).WithDomain(fsDomain).WithCategoryID(categoryId)

	cb, err := connector.New(ctx,
		fsToken,
		fsDomain,
		fsClient,
	)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	opts := make([]connectorbuilder.Opt, 0)
	if cfg.GetBool(field.TicketingField.FieldName) {
		opts = append(opts, connectorbuilder.WithTicketingEnabled())
	}

	c, err := connectorbuilder.NewConnector(ctx, cb, opts...)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return c, nil
}
