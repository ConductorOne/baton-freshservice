package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	linkheader "github.com/tomnomnom/linkheader"
	"go.uber.org/zap"
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

	// https://api.freshservice.com/v2/#rate_limit
	// Rate Limit/Min Ex. List All Agents: 40
	cli, err := uhttp.NewBaseHttpClientWithContext(context.Background(),
		httpClient,
		uhttp.WithRateLimiter(40, time.Minute),
	)
	if err != nil {
		return freshServiceClient, err
	}

	baseUrl := fmt.Sprintf("https://%s.freshservice.com/api/v2", domain)
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

func (f *FreshServiceClient) ListAllUsers(ctx context.Context, opts PageOptions) (*AgentsAPIData, string, any, error) {
	var nextPageToken string = ""
	users, page, statusCode, err := f.GetUsers(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", statusCode, err
	}

	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return users, nextPageToken, statusCode, nil
}

// GetUsers. List All Agents(Users).
// https://api.freshservice.com/v2/#list_all_agents
func (f *FreshServiceClient) GetUsers(ctx context.Context, startPage, limitPerPage string) (*AgentsAPIData, Page, int, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents")
	if err != nil {
		return nil, Page{}, 0, err
	}

	uri, err := url.Parse(agentsUrl)
	if err != nil {
		return nil, Page{}, 0, err
	}

	var res *AgentsAPIData
	page, statusCode, err := f.getListAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, statusCode, err
	}

	return res, page, statusCode, nil
}

func (f *FreshServiceClient) ListAllGroups(ctx context.Context, opts PageOptions) (*GroupsAPIData, string, int, error) {
	var nextPageToken string = ""
	groups, page, statusCode, err := f.GetGroups(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", statusCode, err
	}

	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return groups, nextPageToken, statusCode, nil
}

// GetGroups. List All Agent Groups(Groups).
// https://api.freshservice.com/v2/#view_all_group
func (f *FreshServiceClient) GetGroups(ctx context.Context, startPage, limitPerPage string) (*GroupsAPIData, Page, int, error) {
	groupsUrl, err := url.JoinPath(f.baseUrl, "groups")
	if err != nil {
		return nil, Page{}, 0, err
	}

	uri, err := url.Parse(groupsUrl)
	if err != nil {
		return nil, Page{}, 0, err
	}

	var res *GroupsAPIData
	page, statusCode, err := f.getListAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, statusCode, err
	}

	return res, page, statusCode, nil
}

func (f *FreshServiceClient) getListAPIData(ctx context.Context,
	startPage string,
	limitPerPage string,
	uri *url.URL,
	res any,
) (Page, int, error) {
	var (
		header       http.Header
		err          error
		page         Page
		IsLastPage   = true
		sPage, nPage = "1", "0"
		statusCode   int
	)
	if startPage != "0" {
		sPage = startPage
	}

	limitPage, err := strconv.Atoi(limitPerPage)
	if err != nil {
		return page, 0, err
	}

	if limitPage < 0 || limitPage > 100 {
		limitPerPage = "100"
	}

	setRawQuery(uri, sPage, limitPerPage)
	if header, statusCode, err = f.doRequest(ctx, http.MethodGet, uri.String(), &res, nil); err != nil {
		return page, statusCode, err
	}

	pagingLinks := linkheader.Parse(header.Get("Link"))
	for _, link := range pagingLinks {
		if link.Rel == "next" {
			nextPageUrl, err := url.Parse(link.URL)
			if err != nil {
				return page, 0, err
			}

			nPage = nextPageUrl.Query().Get("page")
			IsLastPage = false
			break
		}
	}

	if !IsLastPage {
		page = Page{
			PreviousPage: &sPage,
			NextPage:     &nPage,
		}
	}

	return page, statusCode, nil
}

func ConvertPageToken(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	return strconv.Atoi(token)
}

// setRawQuery. Set query parameters.
// page : number for the page (inclusive). If not passed, first page is assumed.
// per_page : Number of items to return. If not passed, a page size of 100 is used.
func setRawQuery(uri *url.URL, sPage string, limitPerPage string) {
	q := uri.Query()
	q.Set("per_page", limitPerPage)
	q.Set("page", sPage)
	uri.RawQuery = q.Encode()
}

func (f *FreshServiceClient) ListAllRoles(ctx context.Context, opts PageOptions) (*RolesAPIData, string, int, error) {
	var nextPageToken string = ""
	roles, page, statusCode, err := f.GetRoles(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", statusCode, err
	}

	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return roles, nextPageToken, statusCode, nil
}

// GetRoles. List All Roles.
// https://api.freshservice.com/v2/#view_all_role
func (f *FreshServiceClient) GetRoles(ctx context.Context, startPage, limitPerPage string) (*RolesAPIData, Page, int, error) {
	rolesUrl, err := url.JoinPath(f.baseUrl, "roles")
	if err != nil {
		return nil, Page{}, 0, err
	}

	uri, err := url.Parse(rolesUrl)
	if err != nil {
		return nil, Page{}, 0, err
	}

	var res *RolesAPIData
	page, statusCode, err := f.getListAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, statusCode, err
	}

	return res, page, statusCode, nil
}

// GetGroupDetail. List All Agents in a Group.
// https://api.freshservice.com/v2/#view_a_group
func (f *FreshServiceClient) GetGroupDetail(ctx context.Context, groupId string) (*GroupDetailAPIData, any, error) {
	var (
		statusCode any
		err        error
		res        *GroupDetailAPIData
	)
	groupUrl, err := url.JoinPath(f.baseUrl, "groups", groupId)
	if err != nil {
		return nil, statusCode, err
	}

	if _, statusCode, err = f.doRequest(ctx, http.MethodGet, groupUrl, &res, nil); err != nil {
		return nil, statusCode, err
	}

	if statusCode != http.StatusRequestTimeout {
		return res, statusCode, nil
	}

	return nil, statusCode, nil
}

// UpdateGroupMembers. Update the existing group to add another agent to the group
// https://api.freshservice.com/v2/#update_a_group
func (f *FreshServiceClient) UpdateGroupMembers(ctx context.Context, groupId string, usersId []int64) (any, error) {
	var (
		body            GroupMembers
		res, statusCode any
	)

	body.Members = usersId
	groupUrl, err := url.JoinPath(f.baseUrl, "groups", groupId)
	if err != nil {
		return nil, err
	}

	if _, statusCode, err = f.doRequest(ctx, http.MethodPut, groupUrl, &res, body); err != nil {
		return statusCode, err
	}

	return statusCode, nil
}

// GetAgentDetail. Get agent detail.
// https://api.freshservice.com/v2/#view_an_agent
func (f *FreshServiceClient) GetAgentDetail(ctx context.Context, userId string) (*AgentDetailAPIData, any, error) {
	var (
		statusCode any
		err        error
		res        *AgentDetailAPIData
	)
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents", userId)
	if err != nil {
		return nil, statusCode, err
	}

	if _, statusCode, err = f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
		return nil, statusCode, err
	}

	if statusCode != http.StatusRequestTimeout {
		return res, statusCode, nil
	}

	return nil, statusCode, nil
}

func (f *FreshServiceClient) doRequest(ctx context.Context, method, endpointUrl string, res interface{}, body interface{}) (http.Header, int, error) {
	var (
		resp *http.Response
		err  error
	)
	l := ctxzap.Extract(ctx)
	urlAddress, err := url.Parse(endpointUrl)
	if err != nil {
		return nil, 0, err
	}

	req, err := f.httpClient.NewRequest(ctx,
		method,
		urlAddress,
		uhttp.WithAcceptJSONHeader(),
		uhttp.WithContentTypeJSONHeader(),
		WithSetBasicAuth(f.getToken(), "X"),
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return nil, 0, err
	}

	switch method {
	case http.MethodGet:
		resp, err = f.httpClient.Do(req, uhttp.WithResponse(&res))
		if resp != nil {
			defer resp.Body.Close()
			if err != nil {
				if strings.Contains(err.Error(), "404") || resp.StatusCode == http.StatusRequestTimeout {
					l.Warn(err.Error(),
						zap.Any("error", err),
						zap.String("urlAddress", urlAddress.String()),
					)
				}
			}

			return resp.Header, resp.StatusCode, nil
		}
	case http.MethodPut:
		resp, err = f.httpClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
			if err != nil {
				if strings.Contains(err.Error(), "request timeout") || resp.StatusCode == http.StatusRequestTimeout {
					l.Warn(err.Error(),
						zap.Any("error", err),
						zap.String("urlAddress", urlAddress.String()),
					)
				}
			}

			return resp.Header, resp.StatusCode, nil
		}
	}

	return resp.Header, resp.StatusCode, nil
}

// UpdateAgentRoles. Update an Agent.
// https://api.freshservice.com/v2/#update_an_agent
func (f *FreshServiceClient) UpdateAgentRoles(ctx context.Context, roleIDs []BodyRole, userId string) (any, error) {
	var (
		body            UpdateAgentRoles
		res, statusCode any
	)

	body.Roles = roleIDs
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents", userId)
	if err != nil {
		return nil, err
	}

	if _, statusCode, err = f.doRequest(ctx, http.MethodPut, agentsUrl, &res, body); err != nil {
		return statusCode, err
	}

	return statusCode, nil
}
