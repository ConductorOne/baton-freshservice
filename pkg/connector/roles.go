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
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type roleBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshServiceClient
}

const assignedEntitlement = "assigned"

func (r *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return resourceTypeRole
}

func (r *roleBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var rv []*v2.Resource
	bag, pageToken, err := getToken(pToken, resourceTypeRole)
	if err != nil {
		return nil, "", nil, err
	}

	roles, nextPageToken, annotation, err := r.client.ListAllRoles(ctx, client.PageOptions{
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

	for _, role := range roles.Roles {
		roleCopy := role
		ur, err := roleResource(ctx, &roleCopy, nil)
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

func (r *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	assigmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(agentUserResourceType),
		ent.WithDescription(fmt.Sprintf("Assigned to %s role", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s role %s", resource.DisplayName, assignedEntitlement)),
	}
	rv = append(rv, ent.NewAssignmentEntitlement(resource, assignedEntitlement, assigmentOptions...))

	return rv, "", nil, nil
}

func (r *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (r *roleBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	if principal.Id.ResourceType != agentUserResourceType.Id {
		l.Warn(
			"freshservice-connector: only users can be granted role membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("freshservice-connector: only users can be granted role membership")
	}

	roleId := entitlement.Resource.Id.Resource
	userId := principal.Id.Resource
	roles, annotation, err := r.client.GetAgentDetail(ctx, userId)
	if err != nil {
		return nil, err
	}

	roleId64, err := strconv.ParseInt(roleId, 10, 64)
	if err != nil {
		return nil, err
	}

	var bodyRoles []client.BodyRole
	for _, role := range roles.Agent.Roles {
		bodyRoles = append(bodyRoles, client.BodyRole{
			RoleID:          role.RoleID,
			AssignmentScope: "assigned_items",
		})
	}

	// Adding new role
	bodyRoles = append(bodyRoles, client.BodyRole{
		RoleID:          roleId64,
		AssignmentScope: "assigned_items",
	})

	err = r.client.UpdateAgentRoles(ctx, bodyRoles, userId)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

func (r *roleBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	principal := grant.Principal
	entitlement := grant.Entitlement
	if principal.Id.ResourceType != agentUserResourceType.Id {
		l.Warn(
			"freshservice-connector: only users can have role membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("freshservice-connector: only users can have role membership revoked")
	}

	userId := principal.Id.Resource
	roleId := entitlement.Resource.Id.Resource
	roles, annotation, err := r.client.GetAgentDetail(ctx, userId)
	if err != nil {
		return nil, err
	}

	roleId64, err := strconv.ParseInt(roleId, 10, 64)
	if err != nil {
		return nil, err
	}

	var bodyRoles []client.BodyRole
	for _, role := range roles.Agent.Roles {
		if roleId64 == role.RoleID {
			continue
		}

		bodyRoles = append(bodyRoles, client.BodyRole{
			RoleID:          role.RoleID,
			AssignmentScope: "assigned_items",
		})
	}

	err = r.client.UpdateAgentRoles(ctx, bodyRoles, userId)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

func newRoleBuilder(c *client.FreshServiceClient) *roleBuilder {
	return &roleBuilder{
		resourceType: resourceTypeRole,
		client:       c,
	}
}
