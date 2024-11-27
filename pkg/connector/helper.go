package connector

import (
	"context"
	"strconv"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func userResource(ctx context.Context, user *client.Agents, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	var userStatus v2.UserTrait_Status_Status = v2.UserTrait_Status_STATUS_ENABLED
	// firstName, lastName := splitFullName(user.Contact.Name)
	profile := map[string]interface{}{
		"user_id":    user.ID,
		"login":      user.Email,
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"email":      user.Email,
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
	}

	displayName := user.FirstName + " " + user.LastName
	if user.FirstName == "" {
		displayName = user.Email
	}

	ret, err := rs.NewUserResource(
		displayName,
		userResourceType,
		user.ID,
		userTraits,
		rs.WithParentResourceID(parentResourceID))
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// Create a new connector resource for FreshService.
func groupResource(ctx context.Context, group *client.Group, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"group_id":   group.ID,
		"group_name": group.Name,
		"group_type": group.Type,
	}
	groupTraitOptions := []rs.GroupTraitOption{rs.WithGroupProfile(profile)}
	resource, err := rs.NewGroupResource(
		group.Name,
		groupResourceType,
		group.ID,
		groupTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}

func titleCase(s string) string {
	titleCaser := cases.Title(language.English)

	return titleCaser.String(s)
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
