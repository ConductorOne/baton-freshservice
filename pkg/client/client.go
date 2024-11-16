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

func (f *FreshServiceClient) ListAllUsers(ctx context.Context, opts PageOptions) (*AgentsAPIData, string, error) {
	var nextPageToken string = ""
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
func (f *FreshServiceClient) GetUsers(ctx context.Context, startPage, limitPerPage string) (*AgentsAPIData, Page, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents")
	if err != nil {
		return nil, Page{}, err
	}

	uri, err := url.Parse(agentsUrl)
	if err != nil {
		return nil, Page{}, err
	}

	var res *AgentsAPIData
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

func (f *FreshServiceClient) ListAllGroups(ctx context.Context, opts PageOptions) (*GroupsAPIData, string, error) {
	var nextPageToken string = ""
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
func (f *FreshServiceClient) GetGroups(ctx context.Context, startPage, limitPerPage string) (*GroupsAPIData, Page, error) {
	groupsUrl, err := url.JoinPath(f.baseUrl, "admin", "groups")
	if err != nil {
		return nil, Page{}, err
	}

	uri, err := url.Parse(groupsUrl)
	if err != nil {
		return nil, Page{}, err
	}

	var res *GroupsAPIData
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
	)
	if startPage != "0" {
		sPage = startPage
	}

	setRawQuery(uri, sPage, limitPerPage)
	if err, header, _ = f.doRequest(ctx, http.MethodGet, uri.String(), &res, nil); err != nil {
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

func (f *FreshServiceClient) ListAllRoles(ctx context.Context, opts PageOptions) (*RolesAPIData, string, error) {
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
func (f *FreshServiceClient) GetRoles(ctx context.Context, startPage, limitPerPage string) (*RolesAPIData, Page, error) {
	rolesUrl, err := url.JoinPath(f.baseUrl, "roles")
	if err != nil {
		return nil, Page{}, err
	}

	uri, err := url.Parse(rolesUrl)
	if err != nil {
		return nil, Page{}, err
	}

	var res *RolesAPIData
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

// GetGroups. List All Groups.
// https://developers.freshdesk.com/api/#groups
func (f *FreshServiceClient) GetGroupById(ctx context.Context, groupId string) (*Group, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "admin", "groups", groupId)
	if err != nil {
		return nil, err
	}

	var res *Group
	if err, _, _ := f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
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
	if err, _, _ := f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
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

	if err, _, statusCode = f.doRequest(ctx, http.MethodPatch, agentsUrl, &res, body); err != nil {
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

	if err, _, statusCode = f.doRequest(ctx, http.MethodPatch, agentsUrl, &res, body); err != nil {
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
	if err, _, _ = f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
		return nil, err
	}

	return res, nil
}

func (f *FreshServiceClient) doRequest(ctx context.Context, method, endpointUrl string, res interface{}, body interface{}) (error, http.Header, any) {
	var (
		resp *http.Response
		err  error
	)
	urlAddress, err := url.Parse(endpointUrl)
	if err != nil {
		return err, nil, nil
	}

	req, err := f.httpClient.NewRequest(ctx,
		method,
		urlAddress,
		uhttp.WithAcceptJSONHeader(),
		WithSetBasicAuth(f.getToken(), "X"),
		uhttp.WithJSONBody(body),
	)
	if err != nil {
		return err, nil, nil
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
		return err, nil, nil
	}

	return nil, resp.Header, resp.StatusCode
}

// GetAccount. View Account.
// https://developers.freshdesk.com/api/#account
func (f *FreshServiceClient) GetAccount(ctx context.Context) (*AccountAPIData, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "account")
	if err != nil {
		return nil, err
	}

	var res *AccountAPIData
	if err, _, _ := f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
		return nil, err
	}

	return res, nil
}

// UpdateAgentRoles. Update an Agent.
func (f *FreshServiceClient) UpdateAgentRoles(ctx context.Context, roleIDs []int64, userId string) (any, error) {
	var (
		body struct {
			RoleIDs []int64 `json:"role_ids"`
		}
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

	if err, _, statusCode = f.doRequest(ctx, http.MethodPatch, agentsUrl, &res, body); err != nil {
		return statusCode, err
	}

	return statusCode, nil
}
