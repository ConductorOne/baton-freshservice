package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
	if err, _ := f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
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
	if err, _ := f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
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
	if err, _ := f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
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
	if err, _ := f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
		return nil, err
	}

	return res, nil
}

// GetAgentsByGroupId. List All Agents in a Group.
// https://developers.freshdesk.com/api/#groups
func (f *FreshServiceClient) GetGroupDetail(ctx context.Context, groupId string) (*[]GroupRoles, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "admin", "groups", groupId, "agents")
	if err != nil {
		return nil, err
	}

	var res *[]GroupRoles
	if err, _ := f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
		return nil, err
	}

	return res, nil
}

func (f *FreshServiceClient) AddAgentToGroup(ctx context.Context, groupId, userId string) (any, error) {
	var (
		body struct {
			Agents []struct {
				ID int `json:"id"`
			} `json:"agents"`
		}
		payload         = []byte(fmt.Sprintf(`{ "agents":[{"id": %s}] }`, userId))
		res, statusCode any
	)

	err := json.Unmarshal(payload, &body)
	if err != nil {
		return nil, err
	}

	agentsUrl, err := url.JoinPath(f.baseUrl, "admin", "groups", groupId, "agents")
	if err != nil {
		return nil, err
	}

	if err, statusCode = f.doRequest(ctx, http.MethodPatch, agentsUrl, &res, body); err != nil {
		return statusCode, err
	}

	return statusCode, nil
}

func (f *FreshServiceClient) RemoveAgentFromGroup(ctx context.Context, groupId, userId string) (any, error) {
	var (
		body struct {
			Agents []struct {
				ID      int  `json:"id"`
				Deleted bool `json:"deleted"`
			} `json:"agents"`
		}
		payload         = []byte(fmt.Sprintf(`{ "agents":[{"id": %s, "deleted": true}] }`, userId))
		res, statusCode any
	)

	err := json.Unmarshal(payload, &body)
	if err != nil {
		return nil, err
	}

	agentsUrl, err := url.JoinPath(f.baseUrl, "admin", "groups", groupId, "agents")
	if err != nil {
		return nil, err
	}

	if err, statusCode = f.doRequest(ctx, http.MethodPatch, agentsUrl, &res, body); err != nil {
		return statusCode, err
	}

	return statusCode, nil
}

// GetAgentDetail. Get agent detail.
func (f *FreshServiceClient) GetAgentDetail(ctx context.Context, userId string) (*AgentDetailsAPIData, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents", userId)
	if err != nil {
		return nil, err
	}

	var res *AgentDetailsAPIData
	if err, _ = f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
		return nil, err
	}

	return res, nil
}

func (f *FreshServiceClient) doRequest(ctx context.Context, method, endpointUrl string, res interface{}, body interface{}) (error, any) {
	var (
		resp *http.Response
		err  error
	)
	urlAddress, err := url.Parse(endpointUrl)
	if err != nil {
		return err, nil
	}

	req, err := f.httpClient.NewRequest(ctx,
		method,
		urlAddress,
		uhttp.WithAcceptJSONHeader(),
		WithSetBasicAuth(f.getToken(), "X"),
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return err, nil
	}

	switch method {
	case http.MethodGet:
		resp, err = f.httpClient.Do(req, uhttp.WithResponse(&res))
		defer resp.Body.Close()
	case http.MethodPatch:
		resp, err = f.httpClient.Do(req)
		defer resp.Body.Close()
	}

	if err != nil {
		return err, nil
	}

	return nil, resp.StatusCode
}
