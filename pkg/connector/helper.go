package connector

import (
	"context"
	"strconv"
	"strings"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func userResource(ctx context.Context, user *client.Agent, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	var userStatus v2.UserTrait_Status_Status = v2.UserTrait_Status_STATUS_ENABLED
	firstName, lastName := splitFullName(user.Contact.Name)
	profile := map[string]interface{}{
		"login":      user.Contact.Email,
		"first_name": firstName,
		"last_name":  lastName,
		"email":      user.Contact.Email,
		"user_id":    user.ID,
		"type":       user.Type,
	}

	switch !user.Deactivated {
	case true:
		userStatus = v2.UserTrait_Status_STATUS_ENABLED
	case false:
		userStatus = v2.UserTrait_Status_STATUS_DISABLED
	}

	userTraits := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithStatus(userStatus),
		rs.WithUserLogin(user.Contact.Email),
		rs.WithEmail(user.Contact.Email, true),
	}

	displayName := user.Contact.Name
	if user.Contact.Name == "" {
		displayName = user.Contact.Email
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

// splitFullName returns firstName and lastName.
func splitFullName(name string) (string, string) {
	names := strings.SplitN(name, " ", 2)
	var firstName, lastName string

	switch len(names) {
	case 1:
		firstName = names[0]
	case 2:
		firstName = names[0]
		lastName = names[1]
	}

	return firstName, lastName
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

func accountResource(ctx context.Context, account *client.AccountAPIData, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	var opts []rs.ResourceOption
	profile := map[string]interface{}{
		"organisation_id":          account.OrganisationID,
		"organisation_name":        account.OrganisationName,
		"account_id":               account.AccountID,
		"account_name":             account.AccountName,
		"account_domain":           account.AccountDomain,
		"contact_person_firstname": account.ContactPerson.Firstname,
		"contact_person_lastname":  account.ContactPerson.Lastname,
		"contact_person_email":     account.ContactPerson.Email,
		"tier_type":                account.TierType,
		"data_center":              account.DataCenter,
		"timezone":                 account.Timezone,
	}

	accountTraitOptions := []rs.AppTraitOption{
		rs.WithAppProfile(profile),
	}

	opts = append(opts, rs.WithAppTrait(accountTraitOptions...))
	resource, err := rs.NewResource(
		account.AccountName,
		accountResourceType,
		account.AccountID,
		opts...,
	)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
