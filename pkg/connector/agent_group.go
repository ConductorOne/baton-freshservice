package connector

import (
	"context"
	"fmt"

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

const memberEntitlement = "member"

func (g *groupBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return g.resourceType
}

// List returns all the groups from the database as resource objects.
// Groups include a GroupTrait because they are the 'shape' of a standard group.
func (g *groupBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource
	bag, pageToken, err := getToken(pToken, agentGroupResourceType)
	if err != nil {
		return nil, "", nil, err
	}

	groups, nextPageToken, annotation, err := g.client.ListAllAgentGroups(ctx, client.PageOptions{
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

	for _, group := range groups.Groups {
		groupCopy := group
		ur, err := agentGroupResource(ctx, &groupCopy, nil)
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

func (g *groupBuilder) Entitlements(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	options := []ent.EntitlementOption{
		ent.WithGrantableTo(agentUserResourceType),
		ent.WithDescription(fmt.Sprintf("Access to %s group in FreshService", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s Group %s", resource.DisplayName, memberEntitlement)),
	}
	rv = append(rv, ent.NewAssignmentEntitlement(resource, memberEntitlement, options...))

	return rv, "", nil, nil
}

func (g *groupBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var (
		rv []*v2.Grant
		gr *v2.Grant
	)
	groupDetail, annotation, err := g.client.GetGroupDetail(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	for _, agent := range groupDetail.Group.Members {
		userId := &v2.ResourceId{
			ResourceType: agentUserResourceType.Id,
			Resource:     fmt.Sprintf("%d", agent),
		}
		gr = grant.NewGrant(resource, memberEntitlement, userId)
		rv = append(rv, gr)
	}

	return rv, "", annotation, nil
}

func (g *groupBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	if principal.Id.ResourceType != agentUserResourceType.Id {
		l.Warn(
			"freshservice-connector: only users can be granted group membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("freshservice-connector: only users can be granted group membership")
	}

	groupId := entitlement.Resource.Id.Resource
	userId := principal.Id.Resource
	groupDetail, annotation, err := g.client.GetGroupDetail(ctx, groupId)
	if err != nil {
		return nil, err
	}

	user, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		return nil, err
	}

	groupDetail.Group.Members = append(groupDetail.Group.Members, user)
	_, err = g.client.UpdateAgentGroupMembers(ctx, groupId, groupDetail.Group.Members)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

func (g *groupBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	principal := grant.Principal
	entitlement := grant.Entitlement
	if principal.Id.ResourceType != agentUserResourceType.Id {
		l.Warn(
			"freshservice-connector: only users can have group membership revoked",
			zap.String("principal_id", principal.Id.String()),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, fmt.Errorf("freshservice-connector: only users can have group membership revoked")
	}

	userId := principal.Id.Resource
	groupId := entitlement.Resource.Id.Resource
	groupDetail, annotation, err := g.client.GetGroupDetail(ctx, groupId)
	if err != nil {
		return nil, err
	}

	user, err := strconv.ParseInt(userId, 10, 64)
	if err != nil {
		return nil, err
	}

	var members []int64
	for _, member := range groupDetail.Group.Members {
		if member == user {
			continue
		}

		members = append(members, member)
	}

	_, err = g.client.UpdateAgentGroupMembers(ctx,
		groupId,
		members,
	)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

func newGroupBuilder(c *client.FreshServiceClient) *groupBuilder {
	return &groupBuilder{
		resourceType: agentGroupResourceType,
		client:       c,
	}
}
