package connector

import (
	"context"
	"strconv"
	"strings"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

func requesterUserResource(ctx context.Context, user *client.Requesters, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	userStatus := v2.UserTrait_Status_STATUS_ENABLED
	profile := map[string]interface{}{
		"user_id":    user.ID,
		"login":      user.PrimaryEmail,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"email":      user.PrimaryEmail,
		"is_agent":   false,
	}

	switch user.Active {
	case true:
		userStatus = v2.UserTrait_Status_STATUS_ENABLED
	case false:
		userStatus = v2.UserTrait_Status_STATUS_DISABLED
	}

	userTraits := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithStatus(userStatus),
		rs.WithUserLogin(user.PrimaryEmail),
		rs.WithEmail(user.PrimaryEmail, true),
	}

	displayName := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if displayName == "" {
		displayName = user.PrimaryEmail
	}

	ret, err := rs.NewUserResource(
		displayName,
		requesterResourceType,
		user.ID,
		userTraits,
		rs.WithParentResourceID(parentResourceID))
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func agentResource(ctx context.Context, user *client.Agent, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	userStatus := v2.UserTrait_Status_STATUS_ENABLED
	profile := map[string]interface{}{
		"user_id":    user.ID,
		"login":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"email":      user.Email,
		"is_agent":   true,
	}

	switch user.Active {
	case true:
		userStatus = v2.UserTrait_Status_STATUS_ENABLED
	case false:
		userStatus = v2.UserTrait_Status_STATUS_DISABLED
	}

	userTraits := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithStatus(userStatus),
		rs.WithUserLogin(user.Email),
		rs.WithEmail(user.Email, true),
		rs.WithLastLogin(user.LastLoginAt),
	}

	displayName := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if displayName == "" {
		displayName = user.Email
	}

	ret, err := rs.NewUserResource(
		displayName,
		agentUserResourceType,
		user.ID,
		userTraits,
		rs.WithParentResourceID(parentResourceID))
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// Create a new connector resource for FreshService.
func agentGroupResource(ctx context.Context, group *client.AgentGroup, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"group_id":   group.ID,
		"group_name": group.Name,
	}
	groupTraitOptions := []rs.GroupTraitOption{rs.WithGroupProfile(profile)}
	resource, err := rs.NewGroupResource(
		group.Name,
		agentGroupResourceType,
		group.ID,
		groupTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func unmarshalSkipToken(token *pagination.Token) (int32, *pagination.Bag, error) {
	b := &pagination.Bag{}
	err := b.Unmarshal(token.Token)
	if err != nil {
		return 0, nil, err
	}
	current := b.Current()
	skip := int32(0)
	if current != nil && current.Token != "" {
		skip64, err := strconv.ParseInt(current.Token, 10, 32)
		if err != nil {
			return 0, nil, err
		}
		skip = int32(skip64)
	}
	return skip, b, nil
}

func ConvertPageToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	return strconv.Atoi(token)
}

func roleResource(ctx context.Context, role *client.Roles, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"id":          role.ID,
		"name":        role.Name,
		"description": role.Description,
		"role_type":   role.RoleType,
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

func getToken(pToken *pagination.Token, resourceType *v2.ResourceType) (*pagination.Bag, int, error) {
	var pageToken int
	_, bag, err := unmarshalSkipToken(pToken)
	if err != nil {
		return bag, 0, err
	}

	if bag.Current() == nil {
		bag.Push(pagination.PageState{
			ResourceTypeID: resourceType.Id,
		})
	}

	if bag.Current().Token != "" {
		pageToken, err = strconv.Atoi(bag.Current().Token)
		if err != nil {
			return bag, 0, err
		}
	}

	return bag, pageToken, nil
}

func requesterGroupResource(ctx context.Context, requesterGroup *client.RequesterGroup, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"requester_group_id":   requesterGroup.ID,
		"requester_group_name": requesterGroup.Name,
		"requester_group_type": requesterGroup.Type,
	}
	groupTraitOptions := []rs.GroupTraitOption{rs.WithGroupProfile(profile)}
	resource, err := rs.NewGroupResource(
		requesterGroup.Name,
		resourceTypeRequesterGroup,
		requesterGroup.ID,
		groupTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
