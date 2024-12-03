package connector

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	"github.com/stretchr/testify/require"
)

var (
	apiKey  = os.Getenv("BATON_API_KEY")
	domain  = os.Getenv("BATON_DOMAIN")
	ctxTest = context.Background()
)

func getClientForTesting(ctx context.Context) (*client.FreshServiceClient, error) {
	fsClient := client.NewClient()
	fsClient.WithBearerToken(apiKey).WithDomain(domain)
	return client.New(ctx, fsClient)
}

func TestUsersBuilderList(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	u := &userBuilder{
		resourceType: userResourceType,
		client:       cliTest,
	}
	var token = "{}"
	for token != "" {
		_, tk, _, err := u.List(ctxTest, &v2.ResourceId{}, &pagination.Token{
			Token: token,
		})
		require.Nil(t, err)
		token = tk
	}
}

func TestGroupsBuilderList(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	g := &groupBuilder{
		resourceType: groupResourceType,
		client:       cliTest,
	}
	var token = "{}"
	for token != "" {
		_, tk, _, err := g.List(ctxTest, &v2.ResourceId{}, &pagination.Token{
			Token: token,
		})
		require.Nil(t, err)
		token = tk
	}
}

func TestRolesBuilderList(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	r := &roleBuilder{
		resourceType: resourceTypeRole,
		client:       cliTest,
	}

	var token = "{}"
	for token != "" {
		_, tk, _, err := r.List(ctxTest, &v2.ResourceId{}, &pagination.Token{
			Token: token,
		})
		require.Nil(t, err)
		token = tk
	}
}

func TestRoleGrants(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	r := &roleBuilder{
		resourceType: resourceTypeRole,
		client:       cliTest,
	}

	var token = "{}"
	for token != "" {
		_, tk, _, err := r.Grants(ctxTest, &v2.Resource{
			Id: &v2.ResourceId{
				ResourceType: resourceTypeRole.Id,
				Resource:     "33000064439",
			},
		}, &pagination.Token{
			Token: token,
		})
		require.Nil(t, err)
		token = tk
	}
}

func parseEntitlementID(id string) (*v2.ResourceId, []string, error) {
	parts := strings.Split(id, ":")
	// Need to be at least 3 parts type:entitlement_id:slug
	if len(parts) < 3 || len(parts) > 3 {
		return nil, nil, fmt.Errorf("okta-connector: invalid resource id")
	}

	resourceId := &v2.ResourceId{
		ResourceType: parts[0],
		Resource:     strings.Join(parts[1:len(parts)-1], ":"),
	}

	return resourceId, parts, nil
}

func getGroupForTesting(ctxTest context.Context, id string, name, description string) (*v2.Resource, error) {
	num, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	return groupResource(ctxTest, &client.Group{
		ID:          int64(num),
		Name:        name,
		Description: description,
	}, nil)
}

func getEntitlementForTesting(resource *v2.Resource, resourceDisplayName, entitlement string) *v2.Entitlement {
	options := []ent.EntitlementOption{
		ent.WithGrantableTo(userResourceType),
		ent.WithDisplayName(fmt.Sprintf("%s resource %s", resourceDisplayName, entitlement)),
		ent.WithDescription(fmt.Sprintf("%s of %s freshservice", entitlement, resourceDisplayName)),
	}

	return ent.NewAssignmentEntitlement(resource, entitlement, options...)
}

func TestGroupGrants(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	d := &groupBuilder{
		resourceType: groupResourceType,
		client:       cliTest,
	}
	_, _, _, err = d.Grants(ctxTest, &v2.Resource{
		Id: &v2.ResourceId{ResourceType: groupResourceType.Id, Resource: "33000063690"},
	}, &pagination.Token{})
	require.Nil(t, err)
}

func TestGroupGrant(t *testing.T) {
	var roleEntitlement string
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	grantEntitlement := "group:33000063690:member"
	grantPrincipalType := "user"
	grantPrincipal := "33000161832"
	_, data, err := parseEntitlementID(grantEntitlement)
	require.Nil(t, err)
	require.NotNil(t, data)

	roleEntitlement = data[2]
	resource, err := getGroupForTesting(ctxTest, data[1], "local_group", "test")
	require.Nil(t, err)

	entitlement := getEntitlementForTesting(resource, grantPrincipalType, roleEntitlement)
	g := &groupBuilder{
		resourceType: resourceTypeRole,
		client:       cliTest,
	}
	_, err = g.Grant(ctxTest, &v2.Resource{
		Id: &v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     grantPrincipal,
		},
	}, entitlement)
	require.Nil(t, err)
}

func TestGroupRevoke(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	grantId := "group:33000063690:member:user:33000161832"
	data := strings.Split(grantId, ":")
	principalID := &v2.ResourceId{ResourceType: userResourceType.Id, Resource: data[4]}
	resource, err := getGroupForTesting(ctxTest, data[1], "local_group", "test")
	require.Nil(t, err)

	gr := grant.NewGrant(resource, "member", principalID)
	require.NotNil(t, gr)

	g := &groupBuilder{
		resourceType: resourceTypeRole,
		client:       cliTest,
	}
	_, err = g.Revoke(ctxTest, gr)
	require.Nil(t, err)
}

func TestRoleGrant(t *testing.T) {
	var roleEntitlement string
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	grantEntitlement := "role:156000506894:assigned"
	grantPrincipalType := "user"
	grantPrincipal := "156001115279"
	_, data, err := parseEntitlementID(grantEntitlement)
	require.Nil(t, err)
	require.NotNil(t, data)

	roleEntitlement = data[2]
	resource, err := getRoleForTesting(ctxTest, data[1], "role_agent", "test")
	require.Nil(t, err)

	entitlement := getEntitlementForTesting(resource, grantPrincipalType, roleEntitlement)
	r := &roleBuilder{
		resourceType: resourceTypeRole,
		client:       cliTest,
	}
	_, err = r.Grant(ctxTest, &v2.Resource{
		Id: &v2.ResourceId{
			ResourceType: userResourceType.Id,
			Resource:     grantPrincipal,
		},
	}, entitlement)
	require.Nil(t, err)
}

func getRoleForTesting(ctxTest context.Context, id string, name, description string) (*v2.Resource, error) {
	num, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	return roleResource(ctxTest, &client.Roles{
		ID:          int64(num),
		Name:        name,
		Description: description,
	}, nil)
}

func TestUserGrants(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	u := &userBuilder{
		resourceType: groupResourceType,
		client:       cliTest,
	}
	_, _, _, err = u.Grants(ctxTest, &v2.Resource{
		Id: &v2.ResourceId{ResourceType: groupResourceType.Id, Resource: "33000161861"},
	}, &pagination.Token{})
	require.Nil(t, err)
}
