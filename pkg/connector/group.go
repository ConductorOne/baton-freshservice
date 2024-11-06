package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
)

type groupBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshServiceClient
}

const (
	memberEntitlement = "member"
	adminEntitlement  = "admin"
)

var groupEntitlementAccessLevels = []string{
	memberEntitlement,
	adminEntitlement,
}

func (g *groupBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return g.resourceType
}

// List returns all the groups from the database as resource objects.
// Groups include a GroupTrait because they are the 'shape' of a standard group.
func (g *groupBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	groups, err := g.client.GetGroups(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, group := range *groups {
		groupCopy := group
		ur, err := groupResource(ctx, &groupCopy, nil)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}

	return rv, "", nil, nil
}

func (g *groupBuilder) Entitlements(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	for _, level := range groupEntitlementAccessLevels {
		rv = append(rv, ent.NewPermissionEntitlement(resource, level,
			ent.WithDisplayName(fmt.Sprintf("%s Group %s", resource.DisplayName, titleCase(level))),
			ent.WithDescription(fmt.Sprintf("Access to %s group in FreshService", resource.DisplayName)),
			ent.WithAnnotation(&v2.V1Identifier{
				Id: fmt.Sprintf("group:%s:role:%s", resource.Id.Resource, level),
			}),
			ent.WithGrantableTo(userResourceType),
		))
	}

	return rv, "", nil, nil
}

func (g *groupBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant
	group, err := g.client.GetGroupById(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	for _, agent := range group.AgentIDs {
		userId := &v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     fmt.Sprintf("%d", agent),
		}
		grant := grant.NewGrant(resource, "member", userId)
		rv = append(rv, grant)
	}

	return rv, "", nil, nil
}

func (g *groupBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	return nil, nil
}

func (g *groupBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	return nil, nil
}

func newGroupBuilder(c *client.FreshServiceClient) *groupBuilder {
	return &groupBuilder{
		resourceType: resourceTypeGroup,
		client:       c,
	}
}
