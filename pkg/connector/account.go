package connector

import (
	"context"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
)

type accountBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshServiceClient
}

func (a *accountBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return accountResourceType
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (a *accountBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	account, err := a.client.GetAccount(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	ur, err := accountResource(ctx, account, nil)
	if err != nil {
		return nil, "", nil, err
	}
	rv = append(rv, ur)

	return rv, "", nil, nil
}

// Entitlements always returns an empty slice for users.
func (a *accountBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (a *accountBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func newAccountBuilder(c *client.FreshServiceClient) *accountBuilder {
	return &accountBuilder{
		resourceType: accountResourceType,
		client:       c,
	}
}
