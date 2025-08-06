package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/conductorone/baton-freshservice/pkg/client"
	"github.com/conductorone/baton-freshservice/pkg/config"
	"github.com/conductorone/baton-freshservice/pkg/connector"
	configSchema "github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
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
		config.Config,
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

func getConnector(ctx context.Context, cfg *config.Freshservice) (types.ConnectorServer, error) {
	options := []uhttp.Option{uhttp.WithLogger(true, ctxzap.Extract(ctx))}

	httpClient, err := uhttp.NewClient(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP client failed: %w", err)
	}
	wrapper := uhttp.NewBaseHttpClient(httpClient)
	fsClient := client.NewClient(wrapper)

	l := ctxzap.Extract(ctx)
	fsDomain, err := extractSubdomain(cfg.Domain)
	if err != nil {
		l.Error("error extracting subdomain", zap.Error(err))
		return nil, err
	}

	// Validate the subdomain format
	if strings.Contains(fsDomain, ".") || strings.Contains(fsDomain, "/") || fsDomain == "" {
		return nil, fmt.Errorf("invalid subdomain format: %q - should be just the subdomain portion (e.g., 'company' not 'company.freshservice.com')", fsDomain)
	}

	fsClient = fsClient.WithBearerToken(cfg.ApiKey).WithDomain(fsDomain).WithCategoryID(cfg.CategoryId)

	cb, err := connector.New(ctx,
		cfg.ApiKey,
		fsDomain,
		fsClient,
	)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	opts := make([]connectorbuilder.Opt, 0)
	if cfg.Ticketing {
		opts = append(opts, connectorbuilder.WithTicketingEnabled())
	}

	c, err := connectorbuilder.NewConnector(ctx, cb, opts...)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return c, nil
}
