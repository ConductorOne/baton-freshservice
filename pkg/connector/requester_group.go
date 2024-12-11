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
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type requesterGroupBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshServiceClient
}

func (rg *requesterGroupBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return resourceTypeRequesterGroup
}

func (rg *requesterGroupBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource
	bag, pageToken, err := getToken(pToken, resourceTypeRequesterGroup)
	if err != nil {
		return nil, "", nil, err
	}

	requesterGroups, nextPageToken, annotation, err := rg.client.ListAllRequesterGroups(ctx, client.PageOptions{
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

	for _, requesterGroup := range requesterGroups.RequesterGroups {
		rgCopy := requesterGroup
		ur, err := requesterGroupResource(ctx, &rgCopy, nil)
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

func (rg *requesterGroupBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	options := []ent.EntitlementOption{
		ent.WithGrantableTo(requesterResourceType),
		ent.WithDescription(fmt.Sprintf("Access to %s requester group in FreshService", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s Requester Group %s", resource.DisplayName, memberEntitlement)),
	}
	rv = append(rv, ent.NewAssignmentEntitlement(resource, memberEntitlement, options...))

	return rv, "", nil, nil
}

func (rg *requesterGroupBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var (
		rv []*v2.Grant
		gr *v2.Grant
	)
	groupDetail, annotation, err := rg.client.GetRequesterGroupMembers(ctx, resource.Id.Resource)
	if err != nil {
		return nil, "", nil, err
	}

	for _, requester := range groupDetail.Requesters {
		requesterId := &v2.ResourceId{
			ResourceType: requesterResourceType.Id,
			Resource:     fmt.Sprintf("%d", requester.ID),
		}
		gr = grant.NewGrant(resource, memberEntitlement, requesterId)
		rv = append(rv, gr)
	}

	return rv, "", annotation, nil
}

func (rg *requesterGroupBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	if principal.Id.ResourceType != requesterResourceType.Id {
		l.Warn(
			"freshservice-connector: only requesters can be granted requester group membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("freshservice-connector: only requesters can be granted requester group membership")
	}

	requesterGroupId := entitlement.Resource.Id.Resource
	requesterId := principal.Id.Resource
	annotation, err := rg.client.AddRequesterToRequesterGroup(ctx, requesterGroupId, requesterId)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

func (rg *requesterGroupBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	principal := grant.Principal
	entitlement := grant.Entitlement
	if principal.Id.ResourceType != requesterResourceType.Id {
		l.Warn(
			"freshservice-connector: only requesters can have requester group membership revoked",
			zap.String("principal_id", principal.Id.String()),
			zap.String("principal_type", principal.Id.ResourceType),
		)

		return nil, fmt.Errorf("freshservice-connector: only requesters can have requester group membership revoked")
	}

	requesterId := principal.Id.Resource
	requesterGroupId := entitlement.Resource.Id.Resource
	annotation, err := rg.client.DeleteRequesterFromRequesterGroup(ctx,
		requesterGroupId,
		requesterId,
	)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

func newRequesterGroupBuilder(c *client.FreshServiceClient) *requesterGroupBuilder {
	return &requesterGroupBuilder{
		resourceType: resourceTypeRequesterGroup,
		client:       c,
	}
}
