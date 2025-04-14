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
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/tomnomnom/linkheader"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type FreshServiceClient struct {
	httpClient *uhttp.BaseHttpClient
	auth       *auth
	baseUrl    string
	domain     string
	categoryId string
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

type errorResponse struct {
	MessageContent string `json:"message"`
}

func (er *errorResponse) Message() string {
	return fmt.Sprintf("Error: %s", er.MessageContent)
}

func (f *FreshServiceClient) WithBearerToken(apiToken string) *FreshServiceClient {
	f.auth.bearerToken = apiToken
	return f
}

func (f *FreshServiceClient) WithDomain(domain string) *FreshServiceClient {
	f.domain = domain
	return f
}

func (f *FreshServiceClient) WithCategoryID(categoryID string) *FreshServiceClient {
	f.categoryId = categoryID
	return f
}

func (f *FreshServiceClient) GetCategoryID() string {
	return f.categoryId
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
		categoryId: freshServiceClient.GetCategoryID(),
		auth: &auth{
			bearerToken: clientToken,
		},
	}

	return &fs, nil
}

// https://api.freshservice.com/v2/#list_all_requesters
func (f *FreshServiceClient) ListRequesterUsers(ctx context.Context, opts PageOptions) (*requestersAPIData, string, annotations.Annotations, error) {
	requestersUrl, err := url.JoinPath(f.baseUrl, "requesters")
	if err != nil {
		return nil, "", nil, err
	}
	var res *requestersAPIData
	nextPage, annotation, err := f.getListAPIData(ctx,
		requestersUrl,
		&res,
		WithPage(opts.Page),
		WithPageLimit(opts.PerPage),
	)
	if err != nil {
		return nil, "", nil, err
	}
	return res, nextPage, annotation, nil
}

// https://api.freshservice.com/v2/#list_all_agents
func (f *FreshServiceClient) ListAgentUsers(ctx context.Context, opts PageOptions) (*AgentsAPIData, string, annotations.Annotations, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents")
	if err != nil {
		return nil, "", nil, err
	}

	var res *AgentsAPIData
	nextPage, annotation, err := f.getListAPIData(ctx, agentsUrl, &res, WithPage(opts.Page), WithPageLimit(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}

	return res, nextPage, annotation, nil
}

// https://api.freshservice.com/v2/#view_all_group
func (f *FreshServiceClient) ListAgentGroups(ctx context.Context, opts PageOptions) (*AgentGroupsAPIData, string, annotations.Annotations, error) {
	groupsUrl, err := url.JoinPath(f.baseUrl, "groups")
	if err != nil {
		return nil, "", nil, err
	}

	var res *AgentGroupsAPIData
	nextPage, annotation, err := f.getListAPIData(ctx,
		groupsUrl,
		&res,
		WithPage(opts.Page),
		WithPageLimit(opts.PerPage),
	)
	if err != nil {
		return nil, "", nil, err
	}

	return res, nextPage, annotation, nil
}

func (f *FreshServiceClient) getListAPIData(
	ctx context.Context,
	urlAddress string,
	res any,
	reqOpt ...ReqOpt,
) (string, annotations.Annotations, error) {
	header, annotation, err := f.doRequest(ctx, http.MethodGet, urlAddress, &res, nil, reqOpt...)
	if err != nil {
		return "", annotation, err
	}

	var pageToken string
	pagingLinks := linkheader.Parse(header.Get("Link"))
	for _, link := range pagingLinks {
		if link.Rel == "next" {
			nextPageUrl, err := url.Parse(link.URL)
			if err != nil {
				return "", nil, err
			}
			pageToken = nextPageUrl.Query().Get("page")
			break
		}
	}

	return pageToken, annotation, nil
}

// https://api.freshservice.com/v2/#view_all_role
func (f *FreshServiceClient) ListRoles(ctx context.Context, opts PageOptions) (*RolesAPIData, string, annotations.Annotations, error) {
	rolesUrl, err := url.JoinPath(f.baseUrl, "roles")
	if err != nil {
		return nil, "", nil, err
	}
	var roles *RolesAPIData
	nextPage, annos, err := f.getListAPIData(ctx, rolesUrl, &roles, WithPage(opts.Page), WithPageLimit(opts.PerPage))
	if err != nil {
		return nil, "", nil, err
	}
	return roles, nextPage, annos, nil
}

// GetAgentGroupDetail. List All Agents in a Group.
// https://api.freshservice.com/v2/#view_a_group
func (f *FreshServiceClient) GetAgentGroupDetail(ctx context.Context, groupId string) (*AgentGroupDetailAPIData, annotations.Annotations, error) {
	var res *AgentGroupDetailAPIData
	groupUrl, err := url.JoinPath(f.baseUrl, "groups", groupId)
	if err != nil {
		return nil, nil, err
	}
	_, annotation, err := f.doRequest(ctx, http.MethodGet, groupUrl, &res, nil)
	if err != nil {
		return nil, nil, err
	}
	return res, annotation, nil
}

// UpdateAgentGroupMembers. Update the existing agent group to add another agent to the group
// https://api.freshservice.com/v2/#update_a_group
func (f *FreshServiceClient) UpdateAgentGroupMembers(ctx context.Context, groupId string, usersId []int64) (annotations.Annotations, error) {
	groupUrl, err := url.JoinPath(f.baseUrl, "groups", groupId)
	if err != nil {
		return nil, err
	}

	body := &AgentGroup{Members: usersId}
	_, annotation, err := f.doRequest(ctx, http.MethodPut, groupUrl, nil, body)
	if err != nil {
		return nil, err
	}

	return annotation, nil
}

// GetAgentDetail. Get agent detail.
// https://api.freshservice.com/v2/#view_an_agent
func (f *FreshServiceClient) GetAgentDetail(ctx context.Context, userId string) (*AgentDetailAPIData, annotations.Annotations, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents", userId)
	if err != nil {
		return nil, nil, err
	}

	var res *AgentDetailAPIData
	_, annotation, err := f.doRequest(ctx, http.MethodGet, agentsUrl, &res, nil)
	if err != nil {
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
	reqOptions ...ReqOpt,
) (http.Header, annotations.Annotations, error) {
	var (
		resp *http.Response
		err  error
	)
	urlAddress, err := url.Parse(endpointUrl)
	if err != nil {
		return nil, nil, err
	}
	for _, o := range reqOptions {
		o(urlAddress)
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
		var errRes errorResponse
		doOptions := []uhttp.DoOption{uhttp.WithErrorResponse(&errRes)}

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
func (f *FreshServiceClient) UpdateAgentRoles(ctx context.Context, roleIDs []AgentRole, userId string) (annotations.Annotations, error) {
	agentsUrl, err := url.JoinPath(f.baseUrl, "agents", userId)
	if err != nil {
		return nil, err
	}
	body := &UpdateAgentRoles{Roles: roleIDs}
	_, annos, err := f.doRequest(ctx, http.MethodPut, agentsUrl, nil, body)
	if err != nil {
		return nil, err
	}
	return annos, nil
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

// https://api.freshservice.com/v2/#view_all_requester_group
func (f *FreshServiceClient) ListRequesterGroups(ctx context.Context, opts PageOptions) (*RequesterGroupsAPIData, string, annotations.Annotations, error) {
	requesterGroupsUrl, err := url.JoinPath(f.baseUrl, "requester_groups")
	if err != nil {
		return nil, "", nil, err
	}
	var res *RequesterGroupsAPIData
	nextPage, annotation, err := f.getListAPIData(ctx,
		requesterGroupsUrl,
		&res,
		WithPage(opts.Page),
		WithPageLimit(opts.PerPage),
	)
	if err != nil {
		return nil, "", nil, err
	}
	return res, nextPage, annotation, nil
}

// https://api.freshservice.com/v2/#list_members_of_requester_group
func (f *FreshServiceClient) ListRequesterGroupMembers(ctx context.Context, requesterGroupId string, opts PageOptions) (*requesterGroupMembersAPIData, string, annotations.Annotations, error) {
	requesterGroupMembersUrl, err := url.JoinPath(f.baseUrl, "requester_groups", requesterGroupId, "members")
	if err != nil {
		return nil, "", nil, err
	}
	var res *requesterGroupMembersAPIData
	nextPage, annotation, err := f.getListAPIData(ctx,
		requesterGroupMembersUrl,
		&res,
		WithPage(opts.Page),
		WithPageLimit(opts.PerPage),
	)
	if err != nil {
		return nil, "", nil, err
	}
	return res, nextPage, annotation, nil
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
	groupUrl, err := url.JoinPath(f.baseUrl, "requester_groups", requesterGroupId, "members", requesterId)
	if err != nil {
		return nil, err
	}
	_, annotation, err := f.doRequest(ctx, http.MethodDelete, groupUrl, nil, nil)
	if err != nil {
		return nil, err
	}
	return annotation, nil
}

func (f *FreshServiceClient) GetTicket(ctx context.Context, ticketId string) (*TicketDetails, annotations.Annotations, error) {
	getTicketUrl, err := url.JoinPath(f.baseUrl, "tickets", ticketId)
	if err != nil {
		return nil, nil, err
	}
	var res *TicketResponse
	_, annos, err := f.doRequest(ctx, http.MethodGet, getTicketUrl, &res, nil, WithQueryParam("include", "tags"))
	if err != nil {
		return nil, nil, err
	}
	return res.Ticket, annos, nil
}

// TODO(lauren) this can take workspace_id as query param
func (f *FreshServiceClient) GetTicketFields(ctx context.Context) (*TicketFieldsResponse, error) {
	ticketFormFieldsUrl, err := url.JoinPath(f.baseUrl, "ticket_form_fields")
	if err != nil {
		return nil, err
	}
	var res *TicketFieldsResponse
	_, _, err = f.doRequest(ctx, http.MethodGet, ticketFormFieldsUrl, &res, nil)
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
			ticketStatuses := make([]*v2.TicketStatus, 0, len(tf.Choices))
			for _, choice := range tf.Choices {
				ticketStatuses = append(ticketStatuses, &v2.TicketStatus{
					Id:          strconv.Itoa(choice.ID),
					DisplayName: choice.Value,
				})
			}
			return ticketStatuses, nil
		}
	}
	return nil, errors.New("no ticket statuses found")
}

func (f *FreshServiceClient) GetServiceItem(ctx context.Context, serviceItemID string) (*ServiceItem, error) {
	serviceItemUrl, err := url.JoinPath(f.baseUrl, "service_catalog", "items", serviceItemID)
	if err != nil {
		return nil, err
	}
	var res *ServiceCatalogItemResponse
	_, _, err = f.doRequest(ctx, http.MethodGet, serviceItemUrl, &res, nil)
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

	reqOpts := []ReqOpt{
		WithPage(opts.Page),
		WithPageLimit(opts.PerPage),
	}

	if f.GetCategoryID() != "" {
		reqOpts = append(reqOpts, WithQueryParam("category_id", f.GetCategoryID()))
	}

	var res *ServiceCatalogItemsListResponse
	nextPage, annos, err := f.getListAPIData(ctx,
		ticketFormFieldsUrl,
		&res,
		reqOpts...,
	)
	if err != nil {
		return nil, nil, "", err
	}

	return res, annos, nextPage, nil
}

func (f *FreshServiceClient) CreateServiceRequest(ctx context.Context, serviceCatalogItemID string, payload *ServiceRequestPayload) (*ServiceRequest, annotations.Annotations, error) {
	placeRequestUrl, err := url.JoinPath(f.baseUrl, "service_catalog", "items", serviceCatalogItemID, "place_request")
	if err != nil {
		return nil, nil, err
	}
	var serviceRequestResponse *ServiceRequestResponse
	_, annos, err := f.doRequest(ctx, http.MethodPost, placeRequestUrl, &serviceRequestResponse, payload)
	if err != nil {
		return nil, nil, err
	}
	return serviceRequestResponse.ServiceRequest, annos, nil
}

func (f *FreshServiceClient) UpdateTicket(ctx context.Context, ticketID string, payload *TicketUpdatePayload) (*TicketDetails, annotations.Annotations, error) {
	updateTicketUrl, err := url.JoinPath(f.baseUrl, "tickets", ticketID)
	if err != nil {
		return nil, nil, err
	}
	var ticketUpdateResponse *TicketResponse
	_, annos, err := f.doRequest(ctx, http.MethodPut, updateTicketUrl, &ticketUpdateResponse, payload)
	if err != nil {
		return nil, annos, err
	}
	return ticketUpdateResponse.Ticket, annos, nil
}
