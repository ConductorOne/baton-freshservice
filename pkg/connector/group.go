package connector

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
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
	var (
		pageToken int
		err       error
		rv        []*v2.Resource
	)
	_, bag, err := unmarshalSkipToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	if bag.Current() == nil {
		bag.Push(pagination.PageState{
			ResourceTypeID: groupResourceType.Id,
		})
	}

	if bag.Current().Token != "" {
		pageToken, err = strconv.Atoi(bag.Current().Token)
		if err != nil {
			return nil, "", nil, err
		}
	}

	groups, nextPageToken, err := g.client.ListAllGroups(ctx, client.PageOptions{
		PerPage: ITEMSPERPAGE,
		Page:    pageToken,
	})
	if err != nil {
		return nil, "", nil, err
	}

	err = bag.Next(nextPageToken)
	if err != nil {
		return nil, "", nil, err
	}

	for _, group := range *groups {
		groupCopy := group
		ur, err := groupResource(ctx, &groupCopy, nil)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}

	nextPageToken, err = bag.Marshal()
	if err != nil {
		return nil, "", nil, err
	}

	return rv, nextPageToken, nil, nil
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
	l := ctxzap.Extract(ctx)
	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"freshservice-connector: only users can be granted group membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("freshservice-connector: only users can be granted group membership")
	}

	groupId := entitlement.Resource.Id.Resource
	userId := principal.Id.Resource
	statusCode, err := g.client.AddAgentToGroup(ctx, groupId, userId)
	if err != nil {
		return nil, err
	}

	if http.StatusNoContent == statusCode {
		l.Warn("Membership has been created.",
			zap.String("userId", userId),
			zap.String("groupId", groupId),
		)
	}

	return nil, nil
}

func (g *groupBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	principal := grant.Principal
	entitlement := grant.Entitlement
	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"freshservice-connector: only users can have group membership revoked",
			zap.String("principal_id", principal.Id.String()),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, fmt.Errorf("freshservice-connector: only users can have group membership revoked")
	}

	userId := principal.Id.Resource
	groupId := entitlement.Resource.Id.Resource
	statusCode, err := g.client.RemoveAgentFromGroup(ctx, groupId, userId)
	if err != nil {
		return nil, err
	}

	if http.StatusNoContent == statusCode {
		l.Warn("Membership has been revoked.",
			zap.String("userId", userId),
			zap.String("groupId", groupId),
		)
	}

	return nil, nil
}

func newGroupBuilder(c *client.FreshServiceClient) *groupBuilder {
	return &groupBuilder{
		resourceType: groupResourceType,
		client:       c,
	}
}
