package connector

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

type roleBuilder struct {
	resourceType *v2.ResourceType
	client       *client.FreshServiceClient
}

const (
	assignedEntitlement = "assigned"
	NF                  = -1
)

func (r *roleBuilder) ResourceType(ctx context.Context) *v2.ResourceType {
	return resourceTypeRole
}

func roleResource(ctx context.Context, role *client.Role, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"id":          role.ID,
		"name":        role.Name,
		"description": role.Description,
		"agent_type":  role.AgentType,
	}

	roleTraitOptions := []rs.RoleTraitOption{
		rs.WithRoleProfile(profile),
	}

	resource, err := rs.NewRoleResource(role.Name, resourceTypeRole, role.ID, roleTraitOptions)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func (r *roleBuilder) List(ctx context.Context, parentResourceID *v2.ResourceId, pToken *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	roles, err := r.client.GetRoles(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, role := range *roles {
		roleCopy := role
		ur, err := roleResource(ctx, &roleCopy, nil)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}

	return rv, "", nil, nil
}

func (r *roleBuilder) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	assigmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(userResourceType),
		ent.WithDescription(fmt.Sprintf("Assigned to %s role", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s role %s", resource.DisplayName, assignedEntitlement)),
	}
	rv = append(rv, ent.NewAssignmentEntitlement(resource, assignedEntitlement, assigmentOptions...))

	return rv, "", nil, nil
}

func (r *roleBuilder) Grants(ctx context.Context, resource *v2.Resource, pToken *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var (
		err           error
		rv            []*v2.Grant
		nextPageToken string
	)

	_, bag, err := unmarshalSkipToken(pToken)
	if err != nil {
		return nil, "", nil, err
	}

	if bag.Current() == nil {
		// Push onto stack in reverse
		bag.Push(pagination.PageState{
			ResourceTypeID: groupResourceType.Id,
		})
		bag.Push(pagination.PageState{
			ResourceTypeID: userResourceType.Id,
		})
	}

	if bag.Current().Token != "" {
		nextPageToken = bag.Current().Token
	}

	switch bag.ResourceTypeID() {
	case userResourceType.Id:
		users, err := r.client.GetUsers(ctx)
		if err != nil {
			return nil, "", nil, err
		}

		err = bag.Next(nextPageToken)
		if err != nil {
			return nil, "", nil, err
		}

		for _, user := range *users {
			userId := fmt.Sprintf("%d", user.ID)
			roles, err := r.client.GetAgentDetail(ctx, userId)
			if err != nil {
				return nil, "", nil, err
			}

			rolePos := slices.IndexFunc(roles.RoleIDs, func(c int64) bool {
				roleId := fmt.Sprintf("%d", c)
				return roleId == resource.Id.Resource
			})

			if rolePos != NF {
				userId := &v2.ResourceId{
					ResourceType: userResourceType.Id,
					Resource:     userId,
				}
				grant := grant.NewGrant(resource, assignedEntitlement, userId)
				rv = append(rv, grant)
			}
		}
	case groupResourceType.Id:
		groups, err := r.client.GetGroups(ctx)
		if err != nil {
			return nil, "", nil, err
		}

		err = bag.Next(nextPageToken)
		if err != nil {
			return nil, "", nil, err
		}

		for _, group := range *groups {
			groupId := fmt.Sprintf("%d", group.ID)
			roles, err := r.client.GetGroupDetail(ctx, groupId)
			if err != nil {
				return nil, "", nil, err
			}

			for _, role := range *roles {
				rolePos := slices.IndexFunc(role.RoleIDs, func(c int64) bool {
					roleId := fmt.Sprintf("%d", c)
					return roleId == resource.Id.Resource
				})
				if rolePos != NF {
					groupId := &v2.ResourceId{
						ResourceType: groupResourceType.Id,
						Resource:     groupId,
					}
					grant := grant.NewGrant(resource, assignedEntitlement, groupId)
					rv = append(rv, grant)
				}
			}
		}
	default:
		return nil, "", nil, fmt.Errorf("freshservice connector: invalid grant resource type: %s", bag.ResourceTypeID())
	}

	nextPageToken, err = bag.Marshal()
	if err != nil {
		return nil, "", nil, err
	}

	return rv, nextPageToken, nil, nil
}

func (r *roleBuilder) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"freshservice-connector: only users can be granted role membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("freshservice-connector: only users can be granted role membership")
	}

	roleId := entitlement.Resource.Id.Resource
	userId := principal.Id.Resource
	roles, err := r.client.GetAgentDetail(ctx, userId)
	if err != nil {
		return nil, err
	}

	roleId64, err := strconv.ParseInt(roleId, 10, 64)
	if err != nil {
		return nil, err
	}

	var roleIDs []int64
	for _, role := range roles.RoleIDs {
		if roleId64 == role {
			return nil, fmt.Errorf("freshservice-connector: role already assigned")
		}

		roleIDs = append(roleIDs, role)
	}

	// Adding new role
	roleIDs = append(roleIDs, roleId64)
	statusCode, err := r.client.UpdateAgentRoles(ctx, roleIDs, userId)
	if err != nil {
		return nil, err
	}

	if http.StatusOK == statusCode {
		l.Warn("Membership has been created.",
			zap.String("userId", userId),
			zap.String("roleId", roleId),
		)
	}

	return nil, nil
}

func (r *roleBuilder) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	principal := grant.Principal
	entitlement := grant.Entitlement
	if principal.Id.ResourceType != userResourceType.Id {
		l.Warn(
			"freshservice-connector: only users can have role membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("freshservice-connector: only users can have role membership revoked")
	}

	userId := principal.Id.Resource
	roleId := entitlement.Resource.Id.Resource
	roles, err := r.client.GetAgentDetail(ctx, userId)
	if err != nil {
		return nil, err
	}

	roleId64, err := strconv.ParseInt(roleId, 10, 64)
	if err != nil {
		return nil, err
	}

	var roleIDs []int64
	for _, role := range roles.RoleIDs {
		if roleId64 == role {
			continue
		}

		roleIDs = append(roleIDs, role)
	}

	statusCode, err := r.client.UpdateAgentRoles(ctx, roleIDs, userId)
	if err != nil {
		return nil, err
	}

	if http.StatusOK == statusCode {
		l.Warn("Membership has been revoked.",
			zap.String("userId", userId),
			zap.String("roleId", roleId),
		)
	}

	return nil, nil
}

func newRoleBuilder(c *client.FreshServiceClient) *roleBuilder {
	return &roleBuilder{
		resourceType: resourceTypeRole,
		client:       c,
	}
}