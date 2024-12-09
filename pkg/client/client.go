package client

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	annotations "github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	linkheader "github.com/tomnomnom/linkheader"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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

func (f *FreshServiceClient) GetDomain() string {
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
		domain      = freshServiceClient.GetDomain()
	)
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	cli, err := uhttp.NewBaseHttpClientWithContext(context.Background(),
		httpClient,
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

func (f *FreshServiceClient) ListAllUsers(ctx context.Context, opts PageOptions) (*requestersAPIData, string, annotations.Annotations, error) {
	users, page, annotation, err := f.GetUsers(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return users, *page.NextPage, annotation, nil
}

// GetUsers. List All Agents(Users).
// https://api.freshservice.com/v2/#list_all_agents
func (f *FreshServiceClient) GetUsers(ctx context.Context, startPage, limitPerPage string) (*requestersAPIData, Page, annotations.Annotations, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "requesters")
	if err != nil {
		return nil, Page{}, nil, err
	}

	uri, err := url.Parse(agentsUrl)
	if err != nil {
		return nil, Page{}, nil, err
	}

	q := uri.Query()
	q.Set("include_agents", "true")
	uri.RawQuery = q.Encode()

	var res *requestersAPIData
	page, annotation, err := f.getListAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, nil, err
	}

	return res, page, annotation, nil
}

func (f *FreshServiceClient) ListAllGroups(ctx context.Context, opts PageOptions) (*GroupsAPIData, string, annotations.Annotations, error) {
	groups, page, annotations, err := f.GetGroups(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return groups, *page.NextPage, annotations, nil
}

// GetGroups. List All Agent Groups(Groups).
// https://api.freshservice.com/v2/#view_all_group
func (f *FreshServiceClient) GetGroups(ctx context.Context, startPage, limitPerPage string) (*GroupsAPIData, Page, annotations.Annotations, error) {
	groupsUrl, err := url.JoinPath(f.baseUrl, "groups")
	if err != nil {
		return nil, Page{}, nil, err
	}

	uri, err := url.Parse(groupsUrl)
	if err != nil {
		return nil, Page{}, nil, err
	}

	var res *GroupsAPIData
	page, annotation, err := f.getListAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, nil, err
	}

	return res, page, annotation, nil
}

func (f *FreshServiceClient) getListAPIData(
	ctx context.Context,
	startPage string,
	limitPerPage string,
	uri *url.URL,
	res any,
) (Page, annotations.Annotations, error) {
	var (
		header http.Header
		err    error
		page   = Page{
			PreviousPage: new(string),
			NextPage:     new(string),
		}
		sPage, nPage = "1", "0"
		annotation   annotations.Annotations
	)
	if startPage != "0" {
		sPage = startPage
	}

	limitPage, err := strconv.Atoi(limitPerPage)
	if err != nil {
		return page, nil, err
	}

	if limitPage <= 0 || limitPage > 100 {
		limitPerPage = "100"
	}

	setRawQuery(uri, sPage, limitPerPage)
	if header, annotation, err = f.doRequest(ctx, http.MethodGet, uri.String(), &res, nil); err != nil {
		return page, nil, err
	}

	pagingLinks := linkheader.Parse(header.Get("Link"))
	for _, link := range pagingLinks {
		if link.Rel == "next" {
			nextPageUrl, err := url.Parse(link.URL)
			if err != nil {
				return page, nil, err
			}

			nPage = nextPageUrl.Query().Get("page")
			page = Page{
				PreviousPage: &sPage,
				NextPage:     &nPage,
			}
			break
		}
	}

	return page, annotation, nil
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

func (f *FreshServiceClient) ListAllRoles(ctx context.Context, opts PageOptions) (*RolesAPIData, string, annotations.Annotations, error) {
	roles, page, annotation, err := f.GetRoles(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return roles, *page.NextPage, annotation, nil
}

// GetRoles. List All Roles.
// https://api.freshservice.com/v2/#view_all_role
func (f *FreshServiceClient) GetRoles(ctx context.Context, startPage, limitPerPage string) (*RolesAPIData, Page, annotations.Annotations, error) {
	rolesUrl, err := url.JoinPath(f.baseUrl, "roles")
	if err != nil {
		return nil, Page{}, nil, err
	}

	uri, err := url.Parse(rolesUrl)
	if err != nil {
		return nil, Page{}, nil, err
	}

	var res *RolesAPIData
	page, annotation, err := f.getListAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, nil, err
	}

	return res, page, annotation, nil
}

// GetGroupDetail. List All Agents in a Group.
// https://api.freshservice.com/v2/#view_a_group
func (f *FreshServiceClient) GetGroupDetail(ctx context.Context, groupId string) (*GroupDetailAPIData, annotations.Annotations, error) {
	var (
		err        error
		res        *GroupDetailAPIData
		annotation annotations.Annotations
	)
	groupUrl, err := url.JoinPath(f.baseUrl, "groups", groupId)
	if err != nil {
		return nil, nil, err
	}

	if _, annotation, err = f.doRequest(ctx, http.MethodGet, groupUrl, &res, nil); err != nil {
		return nil, nil, err
	}

	return res, annotation, nil
}

// UpdateGroupMembers. Update the existing group to add another agent to the group
// https://api.freshservice.com/v2/#update_a_group
func (f *FreshServiceClient) UpdateGroupMembers(ctx context.Context, groupId string, usersId []int64) (annotations.Annotations, error) {
	var (
		body       GroupMembers
		res        any
		annotation annotations.Annotations
	)

	body.Members = usersId
	groupUrl, err := url.JoinPath(f.baseUrl, "groups", groupId)
	if err != nil {
		return nil, err
	}

	if _, annotation, err = f.doRequest(ctx, http.MethodPut, groupUrl, &res, body); err != nil {
		return nil, err
	}

	return annotation, nil
}

// GetAgentDetail. Get agent detail.
// https://api.freshservice.com/v2/#view_an_agent
func (f *FreshServiceClient) GetAgentDetail(ctx context.Context, userId string) (*AgentDetailAPIData, annotations.Annotations, error) {
	var (
		err        error
		res        *AgentDetailAPIData
		annotation annotations.Annotations
	)
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents", userId)
	if err != nil {
		return nil, nil, err
	}

	if _, annotation, err = f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil); err != nil {
		return nil, nil, err
	}

	return res, annotation, nil
}

func (f *FreshServiceClient) doRequest(
	ctx context.Context,
	method,
	endpointUrl string,
	res interface{},
	body interface{},
) (http.Header, annotations.Annotations, error) {
	var (
		resp *http.Response
		err  error
	)
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
	case http.MethodGet, http.MethodPut, http.MethodPost:
		doOptions := []uhttp.DoOption{}
		if res != nil {
			doOptions = append(doOptions, uhttp.WithResponse(&res))
		}
		resp, err = f.httpClient.Do(req, doOptions...)
		if resp != nil {
			defer resp.Body.Close()
		}
	case http.MethodDelete:
		resp, err = f.httpClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
	}

	if err != nil {
		return nil, nil, err
	}

	rateLimitData, err := extractRateLimitData(resp)
	if err != nil {
		return nil, nil, err
	}

	annotation := annotations.Annotations{}
	annotation.WithRateLimiting(rateLimitData)

	return resp.Header, annotation, nil
}

// UpdateAgentRoles. Update an Agent.
// https://api.freshservice.com/v2/#update_an_agent
func (f *FreshServiceClient) UpdateAgentRoles(ctx context.Context, roleIDs []BodyRole, userId string) error {
	var (
		body UpdateAgentRoles
		res  any
	)

	body.Roles = roleIDs
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents", userId)
	if err != nil {
		return err
	}

	if _, _, err = f.doRequest(ctx, http.MethodPut, agentsUrl, &res, body); err != nil {
		return err
	}

	return nil
}

// extractRateLimitData returns a set of annotations for rate limiting given the rate limit headers provided by FreshService.
// https://api.freshservice.com/v2/#rate_limit
func extractRateLimitData(response *http.Response) (*v2.RateLimitDescription, error) {
	var (
		err       error
		remaining int64
	)
	if response == nil {
		return nil, fmt.Errorf("freshservice-connector: passed nil response")
	}

	remainingPayload := response.Header.Get("X-RateLimit-Remaining")
	if remainingPayload != "" {
		remaining, err = strconv.ParseInt(remainingPayload, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ratelimit-remaining: %w", err)
		}
	}

	var maxCalls int64
	maxPayload := response.Header.Get("X-RateLimit-Total")
	if maxPayload != "" {
		maxCalls, err = strconv.ParseInt(maxPayload, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ratelimit-total: %w", err)
		}
	}

	var resetAt *timestamppb.Timestamp
	intervalMsPayload := response.Header.Get("Retry-After")
	if intervalMsPayload != "" {
		intervalMs, err := strconv.ParseInt(intervalMsPayload, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse retry-after: %w", err)
		}

		resetAtSeconds := time.Now().Add(time.Duration(intervalMs) * time.Millisecond).Unix()
		resetAt = &timestamppb.Timestamp{Seconds: resetAtSeconds}
	}

	return &v2.RateLimitDescription{
		Limit:     maxCalls,
		Remaining: remaining,
		ResetAt:   resetAt,
	}, nil
}

func (f *FreshServiceClient) ListAllRequesterGroups(ctx context.Context, opts PageOptions) (*RequesterGroupsAPIData, string, annotations.Annotations, error) {
	requesterGroups, page, annotation, err := f.GetRequesterGroups(ctx, strconv.Itoa(opts.Page), strconv.Itoa(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return requesterGroups, *page.NextPage, annotation, nil
}

// GetRequesterGroups. List All Requester Groups.
// https://api.freshservice.com/v2/#view_all_requester_group
func (f *FreshServiceClient) GetRequesterGroups(ctx context.Context, startPage, limitPerPage string) (*RequesterGroupsAPIData, Page, annotations.Annotations, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "requester_groups")
	if err != nil {
		return nil, Page{}, nil, err
	}

	uri, err := url.Parse(agentsUrl)
	if err != nil {
		return nil, Page{}, nil, err
	}

	var res *RequesterGroupsAPIData
	page, annotation, err := f.getListAPIData(ctx,
		startPage,
		limitPerPage,
		uri,
		&res,
	)
	if err != nil {
		return nil, page, nil, err
	}

	return res, page, annotation, nil
}

// GetRequesterGroupMembers. List Requester Group Members.
// https://api.freshservice.com/v2/#list_members_of_requester_group
func (f *FreshServiceClient) GetRequesterGroupMembers(ctx context.Context, requesterGroupId string) (*requesterAPIData, annotations.Annotations, error) {
	var (
		err        error
		res        *requesterAPIData
		annotation annotations.Annotations
	)
	groupUrl, err := url.JoinPath(f.baseUrl, "requester_groups", requesterGroupId, "members")
	if err != nil {
		return nil, nil, err
	}

	if _, annotation, err = f.doRequest(ctx, http.MethodGet, groupUrl, &res, nil); err != nil {
		return nil, nil, err
	}

	return res, annotation, nil
}

// AddRequesterToRequesterGroup. Add Requester to Requester Group.
// https://api.freshservice.com/v2/#add_member_to_requester_group
func (f *FreshServiceClient) AddRequesterToRequesterGroup(
	ctx context.Context,
	requesterGroupId string,
	requesterId string,
) (annotations.Annotations, error) {
	groupUrl, err := url.JoinPath(f.baseUrl, "requester_groups", requesterGroupId, "members", requesterId)
	if err != nil {
		return nil, err
	}
	_, annotation, err := f.doRequest(ctx, http.MethodPost, groupUrl, nil, nil)
	if err != nil {
		return nil, err
	}
	return annotation, nil
}

// DeleteRequesterFromRequesterGroup. Delete Requester from Requester Group.
// https://api.freshservice.com/v2/#delete_member_from_requester_group
func (f *FreshServiceClient) DeleteRequesterFromRequesterGroup(
	ctx context.Context,
	requesterGroupId string,
	requesterId string,
) (annotations.Annotations, error) {
	var (
		res        any
		annotation annotations.Annotations
	)

	groupUrl, err := url.JoinPath(f.baseUrl, "requester_groups", requesterGroupId, "members", requesterId)
	if err != nil {
		return nil, err
	}

	if _, annotation, err = f.doRequest(ctx, http.MethodDelete, groupUrl, &res, nil); err != nil {
		return nil, err
	}

	return annotation, nil
}

func (f *FreshServiceClient) getFetchAPIData(
	ctx context.Context,
	uri *url.URL,
	res any,
) error {
	_, _, err := f.doRequest(ctx, http.MethodGet, uri.String(), &res, nil)
	if err != nil {
		return err
	}
	return nil
}

func (f *FreshServiceClient) GetTicket(ctx context.Context, ticketId string) (*TicketDetails, error) {
	getTicketUrl, err := url.JoinPath(f.baseUrl, "tickets", ticketId)
	if err != nil {
		return nil, err
	}
	uri, err := url.Parse(getTicketUrl)
	if err != nil {
		return nil, err
	}

	v := url.Values{}
	v.Set("include", "tags")

	uri.RawQuery = v.Encode()
	var res *TicketResponse
	err = f.getFetchAPIData(ctx,
		uri,
		&res,
	)
	if err != nil {
		return nil, err
	}
	return res.Ticket, nil
}

// TODO(lauren) this can take workspace_id as query param
func (f *FreshServiceClient) GetTicketFields(ctx context.Context) (*TicketFieldsResponse, error) {
	ticketFormFieldsUrl, err := url.JoinPath(f.baseUrl, "ticket_form_fields")
	if err != nil {
		return nil, err
	}
	uri, err := url.Parse(ticketFormFieldsUrl)
	if err != nil {
		return nil, err
	}
	var res *TicketFieldsResponse
	err = f.getFetchAPIData(ctx, uri, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (f *FreshServiceClient) GetTicketStatuses(ctx context.Context) ([]*v2.TicketStatus, error) {
	ticketFields, err := f.GetTicketFields(ctx)
	if err != nil {
		return nil, err
	}
	for _, tf := range ticketFields.TicketFields {
		if tf.Name == "status" && tf.FieldType == "default_status" {
			ticketStatuses := make([]*v2.TicketStatus, len(tf.Choices))
			for i, choice := range tf.Choices {
				ticketStatuses[i] = &v2.TicketStatus{
					Id:          fmt.Sprint(choice.ID),
					DisplayName: choice.Value,
				}
			}
			return ticketStatuses, nil
		}
	}
	return nil, errors.New("no ticket statuses found")
}

func (f *FreshServiceClient) GetServiceItem(ctx context.Context, serviceItemID string) (*ServiceItem, error) {
	ticketFormFieldsUrl, err := url.JoinPath(f.baseUrl, "service_catalog", "items", serviceItemID)
	if err != nil {
		return nil, err
	}
	uri, err := url.Parse(ticketFormFieldsUrl)
	if err != nil {
		return nil, err
	}
	var res *ServiceCatalogItemResponse
	err = f.getFetchAPIData(ctx, uri, &res)
	if err != nil {
		return nil, err
	}
	return res.ServiceItem, nil
}

// TODO(lauren) this can take workspace_id as query param
// TODO(lauren) this can take category as query param
func (f *FreshServiceClient) ListServiceCatalogItems(ctx context.Context, opts PageOptions) (*ServiceCatalogItemsListResponse, annotations.Annotations, string, error) {
	ticketFormFieldsUrl, err := url.JoinPath(f.baseUrl, "service_catalog", "items")
	if err != nil {
		return nil, nil, "", err
	}
	uri, err := url.Parse(ticketFormFieldsUrl)
	if err != nil {
		return nil, nil, "", err
	}

	var res *ServiceCatalogItemsListResponse
	page, annos, err := f.getListAPIData(ctx,
		strconv.Itoa(opts.Page),
		strconv.Itoa(opts.PerPage),
		uri,
		&res,
	)
	if err != nil {
		return nil, nil, "", err
	}

	var nextPageToken string
	if page.HasNext() {
		nextPageToken = *page.NextPage
	}

	return res, annos, nextPageToken, nil
}

func (f *FreshServiceClient) CreateServiceRequest(ctx context.Context, serviceCatalogItemID string, payload *ServiceRequestPayload) (*ServiceRequest, error) {
	placeRequestUrl, err := url.JoinPath(f.baseUrl, "service_catalog", "items", serviceCatalogItemID, "place_request")
	if err != nil {
		return nil, err
	}
	var serviceRequestResponse *ServiceRequestResponse
	_, _, err = f.doRequest(ctx, http.MethodPost, placeRequestUrl, &serviceRequestResponse, payload)
	if err != nil {
		return nil, err
	}
	return serviceRequestResponse.ServiceRequest, nil
}

func (f *FreshServiceClient) UpdateTicket(ctx context.Context, ticketID string, payload *TicketUpdatePayload) (*TicketDetails, any, error) {
	updateTicketUrl, err := url.JoinPath(f.baseUrl, "tickets", ticketID)
	if err != nil {
		return nil, nil, err
	}
	var ticketUpdateResponse *TicketResponse
	_, statusCode, err := f.doRequest(ctx, http.MethodPut, updateTicketUrl, &ticketUpdateResponse, payload)
	if err != nil {
		return nil, statusCode, err
	}
	return ticketUpdateResponse.Ticket, statusCode, nil
}
