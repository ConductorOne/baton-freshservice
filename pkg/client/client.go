package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
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

	cli, err := uhttp.NewBaseHttpClientWithContext(context.Background(), httpClient)
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
	if opts.HasNotValidPageSize() {
		opts.PerPage = 100
	}

	users, page, statusCode, err := f.GetUsers(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return &AgentsAPIData{}, "", statusCode, err
	}

	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return users, nextPageToken, statusCode, nil
}

// GetUsers. List All Agents(Users).
// https://api.freshservice.com/v2/#list_all_agents
func (f *FreshServiceClient) GetUsers(ctx context.Context, startPage, limitPerPage string) (*AgentsAPIData, Page, any, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents")
	if err != nil {
		return nil, Page{}, nil, err
	}

	uri, err := url.Parse(agentsUrl)
	if err != nil {
		return nil, Page{}, nil, err
	}

	var res *AgentsAPIData
	page, statusCode, err := f.getAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return &AgentsAPIData{}, page, statusCode, err
	}

	return res, page, statusCode, nil
}

func (f *FreshServiceClient) ListAllGroups(ctx context.Context, opts PageOptions) (*GroupsAPIData, string, any, error) {
	var nextPageToken string = ""
	if opts.HasNotValidPageSize() {
		opts.PerPage = 100
	}

	groups, page, statusCode, err := f.GetGroups(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return &GroupsAPIData{}, "", statusCode, err
	}

	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return groups, nextPageToken, statusCode, nil
}

// GetGroups. List All Agent Groups(Groups).
// https://api.freshservice.com/v2/#view_all_group
func (f *FreshServiceClient) GetGroups(ctx context.Context, startPage, limitPerPage string) (*GroupsAPIData, Page, any, error) {
	groupsUrl, err := url.JoinPath(f.baseUrl, "groups")
	if err != nil {
		return nil, Page{}, nil, err
	}

	uri, err := url.Parse(groupsUrl)
	if err != nil {
		return nil, Page{}, nil, err
	}

	var res *GroupsAPIData
	page, statusCode, err := f.getAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return &GroupsAPIData{}, page, statusCode, err
	}

	return res, page, statusCode, nil
}

func (f *FreshServiceClient) getAPIData(ctx context.Context,
	startPage string,
	limitPerPage string,
	uri *url.URL,
	res any,
) (Page, any, error) {
	var (
		header       http.Header
		err          error
		page         Page
		IsLastPage   = true
		sPage, nPage = "1", "0"
		statusCode   any
	)
	if startPage != "0" {
		sPage = startPage
	}

	setRawQuery(uri, sPage, limitPerPage)
	if header, statusCode, err = f.doRequest(ctx, http.MethodGet, uri.String(), &res, nil); err != nil {
		return page, statusCode, err
	}

	if linkUrl, ok := header["Link"]; ok {
		nextLinkUrl, err := getNextLink(linkUrl)
		if err != nil {
			return page, statusCode, err
		}

		params, err := url.ParseQuery(nextLinkUrl.RawQuery)
		if err != nil {
			return page, statusCode, err
		}

		if params.Has("page") {
			nPage = params.Get("page")
			IsLastPage = false
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

// setRawQuery. Set query parameters.
// page : number for the page (inclusive). If not passed, first page is assumed.
// per_page : Number of items to return. If not passed, a page size of 100 is used.
func setRawQuery(uri *url.URL, sPage string, limitPerPage string) {
	q := uri.Query()
	q.Set("per_page", limitPerPage)
	q.Set("page", sPage)
	uri.RawQuery = q.Encode()
}

func (f *FreshServiceClient) ListAllRoles(ctx context.Context, opts PageOptions) (*RolesAPIData, string, any, error) {
	var nextPageToken string = ""
	if opts.HasNotValidPageSize() {
		opts.PerPage = 100
	}

	roles, page, statusCode, err := f.GetRoles(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return &RolesAPIData{}, "", statusCode, err
	}

	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return roles, nextPageToken, statusCode, nil
}

// GetRoles. List All Roles.
// https://api.freshservice.com/v2/#view_all_role
func (f *FreshServiceClient) GetRoles(ctx context.Context, startPage, limitPerPage string) (*RolesAPIData, Page, any, error) {
	rolesUrl, err := url.JoinPath(f.baseUrl, "roles")
	if err != nil {
		return nil, Page{}, nil, err
	}

	uri, err := url.Parse(rolesUrl)
	if err != nil {
		return nil, Page{}, nil, err
	}

	var res *RolesAPIData
	page, statusCode, err := f.getAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return &RolesAPIData{}, page, statusCode, err
	}

	return res, page, statusCode, nil
}

func getNextLink(linkUrl []string) (*url.URL, error) {
	urlStr := strings.Join(linkUrl, "")
	regex := regexp.MustCompile(`[<>;,.!]`)
	result := regex.ReplaceAllString(urlStr, "")
	urlStr = strings.ReplaceAll(result, `rel="next"`, "")
	nextUrl, err := url.Parse(strings.Trim(urlStr, " "))
	return nextUrl, err
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

	return &GroupDetailAPIData{}, statusCode, nil
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

	return &AgentDetailAPIData{}, statusCode, nil
}

func (f *FreshServiceClient) doRequest(ctx context.Context, method, endpointUrl string, res interface{}, body interface{}) (http.Header, any, error) {
	var (
		resp *http.Response
		err  error
	)
	l := ctxzap.Extract(ctx)
	urlAddress, err := url.Parse(endpointUrl)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}

	switch method {
	case http.MethodGet:
		resp, err = f.httpClient.Do(req, uhttp.WithResponse(&res))
		if resp != nil {
			defer resp.Body.Close()
		}
	case http.MethodPut:
		resp, err = f.httpClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
	}

	if err != nil {
		if strings.Contains(err.Error(), "request timeout") || resp.StatusCode == http.StatusRequestTimeout {
			l.Warn("request timeout.",
				zap.Any("error", err),
				zap.String("urlAddress", urlAddress.String()),
			)
			return http.Header{}, http.StatusRequestTimeout, nil
		}

		return http.Header{}, nil, err
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
