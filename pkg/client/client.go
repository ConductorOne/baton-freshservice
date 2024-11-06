package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

type FreshServiceClient struct {
	httpClient *uhttp.BaseHttpClient
	auth       *auth
	baseUrl    string
	domain     string
}

func NewClient() *FreshServiceClient {
	return &FreshServiceClient{
		httpClient: &uhttp.BaseHttpClient{},
		baseUrl:    "",
		auth: &auth{
			bearerToken: "",
		},
	}
}

func (f *FreshServiceClient) WithBearerToken(apiToken string) *FreshServiceClient {
	f.auth.bearerToken = apiToken
	return f
}

func (f *FreshServiceClient) WithDomain(domain string) *FreshServiceClient {
	f.domain = domain
	return f
}

func (f *FreshServiceClient) getToken() string {
	return f.auth.bearerToken
}

func (f *FreshServiceClient) getDomain() string {
	return f.domain
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func WithSetBasicAuth(username, password string) uhttp.RequestOption {
	return uhttp.WithHeader("Authorization", "Basic "+basicAuth(username, password))
}

func isValidUrl(baseUrl string) bool {
	u, err := url.Parse(baseUrl)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func New(ctx context.Context, freshServiceClient *FreshServiceClient) (*FreshServiceClient, error) {
	var (
		clientToken = freshServiceClient.getToken()
		domain      = freshServiceClient.getDomain()
	)
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	cli, err := uhttp.NewBaseHttpClientWithContext(context.Background(), httpClient)
	if err != nil {
		return freshServiceClient, err
	}

	baseUrl := fmt.Sprintf("https://%s.freshdesk.com/api/v2", domain)
	if !isValidUrl(baseUrl) {
		return nil, fmt.Errorf("the url : %s is not valid", baseUrl)
	}

	// bearerToken
	fs := FreshServiceClient{
		httpClient: cli,
		baseUrl:    baseUrl,
		domain:     domain,
		auth: &auth{
			bearerToken: clientToken,
		},
	}

	return &fs, nil
}

// GetUsers. List All Agents.
// https://developers.freshdesk.com/api/#agents
func (f *FreshServiceClient) GetUsers(ctx context.Context) (*AgentsAPIData, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents")
	if err != nil {
		return nil, err
	}

	var res *AgentsAPIData
	if err := f.doRequest(ctx, agentsUrl, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// GetGroups. List All Groups.
// https://developers.freshdesk.com/api/#groups
func (f *FreshServiceClient) GetGroups(ctx context.Context) (*GroupsAPIData, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "admin", "groups")
	if err != nil {
		return nil, err
	}

	var res *GroupsAPIData
	if err := f.doRequest(ctx, agentsUrl, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// GetRoles. List All Roles.
// https://developers.freshdesk.com/api/#roles
func (f *FreshServiceClient) GetRoles(ctx context.Context) (*RolesAPIData, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "roles")
	if err != nil {
		return nil, err
	}

	var res *RolesAPIData
	if err := f.doRequest(ctx, agentsUrl, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// GetGroups. List All Groups.
// https://developers.freshdesk.com/api/#groups
func (f *FreshServiceClient) GetGroupById(ctx context.Context, groupId string) (*Group, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "admin", "groups", groupId)
	if err != nil {
		return nil, err
	}

	var res *Group
	if err := f.doRequest(ctx, agentsUrl, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// GetAgentsByGroupId. List All Agents in a Group.
// https://developers.freshdesk.com/api/#groups
func (f *FreshServiceClient) GetAgentsByGroupId(ctx context.Context, groupId string) (*GroupRolesAPIData, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "admin", "groups", groupId, "agents")
	if err != nil {
		return nil, err
	}

	var res *GroupRolesAPIData
	if err := f.doRequest(ctx, agentsUrl, &res); err != nil {
		return nil, err
	}

	return res, nil
}
func (f *FreshServiceClient) doRequest(ctx context.Context, endpointUrl string, res interface{}) error {
	urlAddress, err := url.Parse(endpointUrl)
	if err != nil {
		return err
	}

	req, err := f.httpClient.NewRequest(ctx,
		http.MethodGet,
		urlAddress,
		uhttp.WithAcceptJSONHeader(),
		WithSetBasicAuth(f.getToken(), "X"),
	)
	if err != nil {
		return err
	}

	resp, err := f.httpClient.Do(req, uhttp.WithResponse(&res))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}
