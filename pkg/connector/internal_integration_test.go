package connector

import (
	"context"
	"os"
	"testing"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
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

func TestUserBuilderList(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	u := &userBuilder{
		resourceType: userResourceType,
		client:       cliTest,
	}

	_, _, _, err = u.List(ctxTest, &v2.ResourceId{}, &pagination.Token{})
	require.Nil(t, err)
}

func TestGroupBuilderList(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	g := &groupBuilder{
		resourceType: resourceTypeGroup,
		client:       cliTest,
	}

	_, _, _, err = g.List(ctxTest, &v2.ResourceId{}, &pagination.Token{})
	require.Nil(t, err)
}

func TestGroupGrants(t *testing.T) {
	if apiKey == "" && domain == "" {
		t.Skip()
	}

	cliTest, err := getClientForTesting(ctxTest)
	require.Nil(t, err)

	d := &groupBuilder{
		resourceType: resourceTypeGroup,
		client:       cliTest,
	}
	_, _, _, err = d.Grants(ctxTest, &v2.Resource{
		Id: &v2.ResourceId{ResourceType: resourceTypeGroup.Id, Resource: "156000164892"},
	}, &pagination.Token{})
	require.Nil(t, err)
}
