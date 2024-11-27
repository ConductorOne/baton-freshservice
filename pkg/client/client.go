package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

	baseUrl := fmt.Sprintf("%s/api/v2", domain)
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

func (f *FreshServiceClient) ListAllUsers(ctx context.Context, opts PageOptions) (*AgentsAPIDataV2, string, error) {
	var nextPageToken string = ""
	if !opts.HasValidPageSize() {
		opts.PerPage = 100
	}

	users, page, err := f.GetUsers(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", err
	}

	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return users, nextPageToken, nil
}

// GetUsers. List All Agents(Users).
// https://developers.freshdesk.com/api/#agents
func (f *FreshServiceClient) GetUsers(ctx context.Context, startPage, limitPerPage string) (*AgentsAPIDataV2, Page, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents")
	if err != nil {
		return nil, Page{}, err
	}

	uri, err := url.Parse(agentsUrl)
	if err != nil {
		return nil, Page{}, err
	}

	var res *AgentsAPIDataV2
	page, err := f.getAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, err
	}

	return res, page, nil
}

func (f *FreshServiceClient) ListAllGroups(ctx context.Context, opts PageOptions) (*GroupsAPIDataV2, string, error) {
	var nextPageToken string = ""
	if !opts.HasValidPageSize() {
		opts.PerPage = 100
	}

	groups, page, err := f.GetGroups(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", err
	}

	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return groups, nextPageToken, nil
}

// GetGroups. List All Agent Groups(Groups).
// https://developers.freshdesk.com/api/#groups
func (f *FreshServiceClient) GetGroups(ctx context.Context, startPage, limitPerPage string) (*GroupsAPIDataV2, Page, error) {
	groupsUrl, err := url.JoinPath(f.baseUrl, "groups")
	if err != nil {
		return nil, Page{}, err
	}

	uri, err := url.Parse(groupsUrl)
	if err != nil {
		return nil, Page{}, err
	}

	var res *GroupsAPIDataV2
	page, err := f.getAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, err
	}

	return res, page, nil
}

func (f *FreshServiceClient) getAPIData(ctx context.Context,
	startPage string,
	limitPerPage string,
	uri *url.URL,
	res any,
) (Page, error) {
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
		if statusCode != http.StatusRequestTimeout {
			return page, nil
		}

		return page, err
	}

	if linkUrl, ok := header["Link"]; ok {
		nextLinkUrl, err := getNextLink(linkUrl)
		if err != nil {
			return page, err
		}

		params, err := url.ParseQuery(nextLinkUrl.RawQuery)
		if err != nil {
			return page, err
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

	return page, nil
}

// setRawQuery. Set query parameters.
// page : number for the page (inclusive). If not passed, first page is assumed.
// per_page : Number of items to return. If not passed, a page size of 30 is used.
func setRawQuery(uri *url.URL, sPage string, limitPerPage string) {
	q := uri.Query()
	q.Set("per_page", limitPerPage)
	q.Set("page", sPage)
	uri.RawQuery = q.Encode()
}

func (f *FreshServiceClient) ListAllRoles(ctx context.Context, opts PageOptions) (*RolesAPIDataV2, string, error) {
	var nextPageToken string = ""
	roles, page, err := f.GetRoles(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", err
	}

	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return roles, nextPageToken, nil
}

// GetRoles. List All Roles.
// https://developers.freshdesk.com/api/#roles
func (f *FreshServiceClient) GetRoles(ctx context.Context, startPage, limitPerPage string) (*RolesAPIDataV2, Page, error) {
	rolesUrl, err := url.JoinPath(f.baseUrl, "roles")
	if err != nil {
		return nil, Page{}, err
	}

	uri, err := url.Parse(rolesUrl)
	if err != nil {
		return nil, Page{}, err
	}

	var res *RolesAPIDataV2
	page, err := f.getAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, err
	}

	return res, page, nil
}

func getNextLink(linkUrl []string) (*url.URL, error) {
	urlStr := strings.Join(linkUrl, "")
	regex := regexp.MustCompile(`[<>;,.!]`)
	result := regex.ReplaceAllString(urlStr, "")
	urlStr = strings.ReplaceAll(result, `rel="next"`, "")
	nextUrl, err := url.Parse(strings.Trim(urlStr, " "))
	return nextUrl, err
}

// GetAgentsByGroupId. List All Agents in a Group.
// https://developers.freshdesk.com/api/#groups
func (f *FreshServiceClient) GetGroupDetail(ctx context.Context, groupId string) (*GroupDetailAPIData, error) {
	var (
		statusCode any
		err        error
		res        *GroupDetailAPIData
	)
	groupUrl, err := url.JoinPath(f.baseUrl, "groups", groupId)
	if err != nil {
		return nil, err
	}

	if _, statusCode, err = f.doRequest(ctx, http.MethodGet, groupUrl, &res, nil); err != nil {
		return nil, err
	}

	if statusCode != http.StatusRequestTimeout {
		return res, nil
	}

	return &GroupDetailAPIData{}, nil
}

func (f *FreshServiceClient) AddAgentToGroup(ctx context.Context, groupId, userId string) (any, error) {
	var (
		body            AddAgentToGroup
		payload         = []byte(fmt.Sprintf(`{ "agents":[{"id": %s}] }`, userId))
		res, statusCode any
	)

	err := json.Unmarshal(payload, &body)
	if err != nil {
		return nil, err
	}

	agentsUrl, err := url.JoinPath(f.baseUrl, "groups", groupId, "agents")
	if err != nil {
		return nil, err
	}

	if _, statusCode, err = f.doRequest(ctx, http.MethodPatch, agentsUrl, &res, body); err != nil {
		return statusCode, err
	}

	return statusCode, nil
}

func (f *FreshServiceClient) RemoveAgentFromGroup(ctx context.Context, groupId, userId string) (any, error) {
	var (
		body            RemoveAgentFromGroup
		payload         = []byte(fmt.Sprintf(`{ "agents":[{"id": %s, "deleted": true}] }`, userId))
		res, statusCode any
	)

	err := json.Unmarshal(payload, &body)
	if err != nil {
		return nil, err
	}

	agentsUrl, err := url.JoinPath(f.baseUrl, "groups", groupId, "agents")
	if err != nil {
		return nil, err
	}

	if _, statusCode, err = f.doRequest(ctx, http.MethodPatch, agentsUrl, &res, body); err != nil {
		return statusCode, err
	}

	return statusCode, nil
}

// GetAgentDetail. Get agent detail.
func (f *FreshServiceClient) GetAgentDetail(ctx context.Context, userId string) (*Agents, error) {
	var (
		statusCode any
		err        error
		res        *Agents
	)
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents", userId)
	if err != nil {
		return nil, err
	}

	if _, statusCode, err = f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
		return nil, err
	}

	if statusCode != http.StatusRequestTimeout {
		return res, nil
	}

	return &Agents{}, nil
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
	case http.MethodPatch:
		resp, err = f.httpClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
	}

	if err != nil {
		if strings.Contains(err.Error(), "request timeout") {
			l.Warn("request timeout.",
				zap.String("urlAddress", urlAddress.String()),
			)
			return nil, http.StatusRequestTimeout, nil
		}

		return nil, nil, err
	}

	return resp.Header, resp.StatusCode, nil
}

// UpdateAgentRoles. Update an Agent.
func (f *FreshServiceClient) UpdateAgentRoles(ctx context.Context, roleIDs []int64, userId string) (any, error) {
	var (
		body            UpdateAgentRoles
		res, statusCode any
	)

	ids, err := json.Marshal(roleIDs)
	if err != nil {
		return nil, err
	}

	payload := []byte(fmt.Sprintf(`{ "role_ids": %s }`, ids))
	err = json.Unmarshal(payload, &body)
	if err != nil {
		return nil, err
	}

	agentsUrl, err := url.JoinPath(f.baseUrl, "agents", userId)
	if err != nil {
		return nil, err
	}

	if _, statusCode, err = f.doRequest(ctx, http.MethodPatch, agentsUrl, &res, body); err != nil {
		return statusCode, err
	}

	return statusCode, nil
}
