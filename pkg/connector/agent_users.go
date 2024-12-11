package connector

import (
	"context"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
)

type agentUserBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshServiceClient
}

func (u *agentUserBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return agentUserResourceType
}

// List returns all the users from the database as resource objects.
// Users include a UserTrait because they are the 'shape' of a standard user.
func (u *agentUserBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource
	bag, pageToken, err := getToken(pToken, agentUserResourceType)
	if err != nil {
		return nil, "", nil, err
	}

	users, nextPageToken, annotation, err := u.client.ListAgentUsers(ctx, client.PageOptions{
		PerPage: pToken.Size,
		Page:    pageToken,
	})
	if err != nil {
		return nil, "", nil, err
	}

	err = bag.Next(nextPageToken)
	if err != nil {
		return nil, "", nil, err
	}

	for _, user := range users.Agents {
		userCopy := user
		ur, err := agentResource(ctx, &userCopy, nil)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}

	nextPageToken, err = bag.Marshal()
	if err != nil {
		return nil, "", nil, err
	}

	return rv, nextPageToken, annotation, nil
}

// Entitlements always returns an empty slice for users.
func (u *agentUserBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grants always returns an empty slice for users since they don't have any entitlements.
func (u *agentUserBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant

	userId := resource.Id.Resource

	agentDetail, annotation, err := u.client.GetAgentDetail(ctx, userId)
	if err != nil {
		return nil, "", nil, err
	}

	for _, role := range agentDetail.Agent.Roles {
		roleRes, err := roleResource(ctx, &client.Roles{
			ID: role.RoleID,
		}, nil)
		if err != nil {
			return nil, "", nil, err
		}

		userId := &v2.ResourceId{
			ResourceType: agentUserResourceType.Id,
			Resource:     userId,
		}
		grant := grant.NewGrant(roleRes, assignedEntitlement, userId)
		rv = append(rv, grant)
	}

	return rv, "", annotation, nil
}

func newAgentUserBuilder(c *client.FreshServiceClient) *agentUserBuilder {
	return &agentUserBuilder{
		resourceType: agentUserResourceType,
		client:       c,
	}
}
