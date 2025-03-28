package connector

import (
	"context"
	"io"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
)

type Connector struct {
	client *client.FreshServiceClient
}

// ResourceSyncers returns a ResourceSyncer for each resource type that should be synced from the upstream service.
func (d *Connector) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		newAgentUserBuilder(d.client),
		newRequesterUserBuilder(d.client),
		newGroupBuilder(d.client),
		newRoleBuilder(d.client),
		newRequesterGroupBuilder(d.client),
	}
}

// Asset takes an input AssetRef and attempts to fetch it using the connector's authenticated http client
// It streams a response, always starting with a metadata object, following by chunked payloads for the asset.
func (d *Connector) Asset(ctx context.Context, asset *v2.AssetRef) (string, io.ReadCloser, error) {
	return "", nil, nil
}

// Metadata returns metadata about the connector.
func (d *Connector) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "FreshService Connector",
		Description: "Connector syncing users, groups, roles and requester groups from FreshService.",
	}, nil
}

// Validate is called to ensure that the connector is properly configured. It should exercise any API credentials
// to be sure that they are valid.
func (d *Connector) Validate(ctx context.Context) (annotations.Annotations, error) {
	return nil, nil
}

// New returns a new instance of the connector.
func New(ctx context.Context, apiKey, domain string, freshServiceClient *client.FreshServiceClient) (*Connector, error) {
	var err error
	if apiKey != "" && domain != "" {
		freshServiceClient, err = client.New(ctx, freshServiceClient)
		if err != nil {
			return nil, err
		}
	}

	return &Connector{
		client: freshServiceClient,
	}, nil
}
