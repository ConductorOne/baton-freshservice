package connector

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/conductorone/baton-freshservice/pkg/client"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
	sdkTicket "github.com/conductorone/baton-sdk/pkg/types/ticket"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ticketUrlFmt = "https://%s.freshservice.com/a/tickets/%d"

func (c *Connector) ListTicketSchemas(ctx context.Context, pt *pagination.Token) ([]*v2.TicketSchema, string, annotations.Annotations, error) {
	var ret []*v2.TicketSchema

	page, err := ConvertPageToken(pt.Token)
	if err != nil {
		return nil, "", nil, err
	}

	serviceCatalogItems, annos, nextPage, err := c.client.ListServiceCatalogItems(ctx, client.PageOptions{
		PerPage: pt.Size,
		Page:    page,
	})
	if err != nil {
		return nil, "", nil, err
	}

	ticketStatuses, err := c.client.GetTicketStatuses(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	for _, serviceItem := range serviceCatalogItems.ServiceItems {
		if serviceItem.Deleted || serviceItem.Visibility == client.ServiceItemVisibilityDraft {
			continue
		}
		ticketSchema, err := c.schemaForServiceCatalogItem(ctx, strconv.FormatInt(serviceItem.DisplayID, 10), ticketStatuses)
		if err != nil {
			return nil, "", nil, err
		}
		ticketSchema.Statuses = ticketStatuses
		ret = append(ret, ticketSchema)
	}

	return ret, nextPage, annos, nil
}

func (c *Connector) GetTicket(ctx context.Context, ticketId string) (*v2.Ticket, annotations.Annotations, error) {
	ticket, annos, err := c.client.GetTicket(ctx, ticketId)
	if err != nil {
		return nil, nil, err
	}

	domain := c.client.GetDomain()
	ticketUrl := fmt.Sprintf(ticketUrlFmt, domain, ticket.ID)

	return &v2.Ticket{
		Id:          strconv.Itoa(ticket.ID),
		DisplayName: ticket.Subject,
		Description: ticket.DescriptionText,
		Status:      &v2.TicketStatus{Id: strconv.Itoa(ticket.Status)},
		Labels:      ticket.Tags,
		Url:         ticketUrl,
		CreatedAt:   timestamppb.New(ticket.CreatedAt),
		UpdatedAt:   timestamppb.New(ticket.UpdatedAt),
	}, annos, nil
}

func (c *Connector) CreateTicket(ctx context.Context, ticket *v2.Ticket, schema *v2.TicketSchema) (*v2.Ticket, annotations.Annotations, error) {
	ticketFields := ticket.GetCustomFields()
	ticketOptions := []client.CustomFieldOption{}
	for id, cf := range schema.GetCustomFields() {
		val, err := sdkTicket.GetCustomFieldValueOrDefault(ticketFields[id])
		if err != nil {
			return nil, nil, err
		}
		// The ticket doesn't have this key set, so we skip it
		if val == nil {
			continue
		}
		ticketOptions = append(ticketOptions, client.WithCustomField(cf.GetId(), val))
	}

	valid, err := sdkTicket.ValidateTicket(ctx, schema, ticket)
	if err != nil {
		return nil, nil, err
	}
	if !valid {
		return nil, nil, errors.New("error: unable to create ticket, ticket is invalid")
	}

	var requestedForEmail string
	if ticket.RequestedFor != nil {
		ut, err := resource.GetUserTrait(ticket.RequestedFor)
		if err != nil {
			return nil, nil, err
		}
		requestedForEmail = ut.Login
	}

	// TODO(lauren) set "reporter" in c1 ticket request (set this from c1 created_by)
	var requestedByEmail string
	if ticket.Reporter != nil {
		ut, err := resource.GetUserTrait(ticket.Reporter)
		if err != nil {
			return nil, nil, err
		}
		requestedByEmail = ut.Login
	}

	// Default to current agent if requestedBy is empty
	if requestedByEmail == "" {
		agent, _, err := c.client.GetAgentDetail(ctx, "me")
		if err != nil {
			return nil, nil, err
		}
		requestedByEmail = agent.Agent.Email
	}

	createServiceCatalogRequestPayload := &client.ServiceRequestPayload{
		RequestedFor: requestedForEmail,
		Email:        requestedByEmail,
		Quantity:     1,
	}
	for _, opt := range ticketOptions {
		opt(createServiceCatalogRequestPayload)
	}

	catalogItemID := schema.GetId()

	serviceRequest, annos, err := c.client.CreateServiceRequest(ctx, catalogItemID, createServiceCatalogRequestPayload)
	if err != nil {
		return nil, nil, fmt.Errorf("freshservice-connector: failed to create service request %s: %w", catalogItemID, err)
	}

	updateTicketReq := &client.TicketUpdatePayload{
		Description: ticket.Description,
		Subject:     ticket.DisplayName,
		Tags:        ticket.Labels,
	}

	domain := c.client.GetDomain()
	serviceRequestURL := fmt.Sprintf(ticketUrlFmt, domain, serviceRequest.ID)

	ticketResp := &v2.Ticket{
		Id:           strconv.Itoa(serviceRequest.ID),
		DisplayName:  serviceRequest.Subject,
		Description:  serviceRequest.DescriptionText,
		Status:       &v2.TicketStatus{Id: strconv.Itoa(serviceRequest.Status)},
		Labels:       serviceRequest.Tags,
		Url:          serviceRequestURL,
		CreatedAt:    timestamppb.New(serviceRequest.CreatedAt),
		UpdatedAt:    timestamppb.New(serviceRequest.UpdatedAt),
		RequestedFor: ticket.RequestedFor,
	}

	updatedTicket, annos, err := c.client.UpdateTicket(ctx, ticketResp.Id, updateTicketReq)
	if err != nil {
		return ticketResp, annos, fmt.Errorf("freshservice-connector: failed to update ticket %s: %w", catalogItemID, err)
	}

	ticketResp.DisplayName = updatedTicket.Subject
	ticketResp.Description = updatedTicket.DescriptionText
	ticketResp.Labels = updatedTicket.Tags

	return ticketResp, annos, nil
}

func (c *Connector) GetTicketSchema(ctx context.Context, schemaID string) (*v2.TicketSchema, annotations.Annotations, error) {
	ticketStatuses, err := c.client.GetTicketStatuses(ctx)
	if err != nil {
		return nil, nil, err
	}
	ticketSchema, err := c.schemaForServiceCatalogItem(ctx, schemaID, ticketStatuses)
	if err != nil {
		return nil, nil, err
	}
	return ticketSchema, nil, nil
}

func (c *Connector) BulkCreateTickets(ctx context.Context, request *v2.TicketsServiceBulkCreateTicketsRequest) (*v2.TicketsServiceBulkCreateTicketsResponse, error) {
	tickets := make([]*v2.TicketsServiceCreateTicketResponse, 0)
	for _, ticketReq := range request.GetTicketRequests() {
		reqBody := ticketReq.GetRequest()
		ticketBody := &v2.Ticket{
			DisplayName:  reqBody.GetDisplayName(),
			Description:  reqBody.GetDescription(),
			Status:       reqBody.GetStatus(),
			Labels:       reqBody.GetLabels(),
			CustomFields: reqBody.GetCustomFields(),
			RequestedFor: reqBody.GetRequestedFor(),
		}
		ticket, annos, err := c.CreateTicket(ctx, ticketBody, ticketReq.GetSchema())
		// So we can track the external ticket ref annotation
		annos.Merge(ticketReq.GetAnnotations()...)
		var ticketResp *v2.TicketsServiceCreateTicketResponse
		if err != nil {
			ticketResp = &v2.TicketsServiceCreateTicketResponse{Ticket: ticket, Annotations: annos, Error: err.Error()}
		} else {
			ticketResp = &v2.TicketsServiceCreateTicketResponse{Ticket: ticket, Annotations: annos}
		}
		tickets = append(tickets, ticketResp)
	}
	return &v2.TicketsServiceBulkCreateTicketsResponse{Tickets: tickets}, nil
}

func (c *Connector) BulkGetTickets(ctx context.Context, request *v2.TicketsServiceBulkGetTicketsRequest) (*v2.TicketsServiceBulkGetTicketsResponse, error) {
	tickets := make([]*v2.TicketsServiceGetTicketResponse, 0)
	for _, ticketReq := range request.GetTicketRequests() {
		ticket, annos, err := c.GetTicket(ctx, ticketReq.GetId())
		// So we can track the external ticket ref annotation
		annos.Merge(ticketReq.GetAnnotations()...)
		var ticketResp *v2.TicketsServiceGetTicketResponse
		if err != nil {
			ticketResp = &v2.TicketsServiceGetTicketResponse{Ticket: ticket, Annotations: annos, Error: err.Error()}
		} else {
			ticketResp = &v2.TicketsServiceGetTicketResponse{Ticket: ticket, Annotations: annos}
		}
		tickets = append(tickets, ticketResp)
	}
	return &v2.TicketsServiceBulkGetTicketsResponse{Tickets: tickets}, nil
}

func (c *Connector) schemaForServiceCatalogItem(ctx context.Context, schemaID string, ticketStatuses []*v2.TicketStatus) (*v2.TicketSchema, error) {
	l := ctxzap.Extract(ctx)
	serviceItem, err := c.client.GetServiceItem(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("freshservice-connector: failed to get service item %s: %w", schemaID, err)
	}
	customFields := make(map[string]*v2.TicketCustomField)
	for _, cf := range serviceItem.CustomFields {
		if cf.Deleted {
			continue
		}
		var cfSchema *v2.TicketCustomField
		switch cf.FieldType {
		case "custom_text", "custom_paragraph", "custom_url":
			cfSchema = sdkTicket.StringFieldSchema(cf.Name, cf.Label, cf.Required)
		case "custom_date":
			cfSchema = sdkTicket.TimestampFieldSchema(cf.Name, cf.Label, cf.Required)
		case "custom_checkbox":
			cfSchema = sdkTicket.BoolFieldSchema(cf.Name, cf.Label, cf.Required)
		case "custom_dropdown":
			// Dropdown choices are an array of a string array that has two elements, both are the same text value
			/*
				"choices": [
				  [
				    "Another dropdown first Choice",
				    "Another dropdown first Choice"
				  ],
				  [
				    "Another dropdown second Choice",
				    "Another dropdown second Choice"
				  ]
				],
			*/
			allowedValues := make([]string, 0, len(cf.Choices))
			for _, choice := range cf.Choices {
				allowedValues = append(allowedValues, choice[0])
			}
			cfSchema = sdkTicket.PickStringFieldSchema(cf.Name, cf.Label, cf.Required, allowedValues)
		case "custom_multi_select_dropdown":
			// Multiselect choices are an array of a string array that has two elements, first element is ID and second element is string
			/*
			   "choices": [
			     [
			       "First multi choice",
			       "6a4abac7-4871-4fd3-a6c7-bcbb485550f6"
			     ],
			     [
			       "Second multi choice",
			       "c978633e-dea9-41cb-bbd6-88a86a6afc06"
			     ],
			     [
			       "Third multi choice",
			       "c480afce-d009-4ea8-a471-6c141a64235f"
			     ]
			   ],
			*/
			allowedValues := make([]string, 0, len(cf.Choices))
			for _, choice := range cf.Choices {
				allowedValues = append(allowedValues, choice[0])
			}
			cfSchema = sdkTicket.PickMultipleStringsFieldSchema(cf.Name, cf.Label, cf.Required, allowedValues)
		case "custom_lookup_bigint":
			cfSchema = sdkTicket.StringFieldSchema(cf.Name, cf.Label, cf.Required)
		case "custom_multi_lookup":
			// This must be populated with array of IDs
			cfSchema = sdkTicket.StringsFieldSchema(cf.Name, cf.Label, cf.Required)
		case "custom_decimal", "custom_number":
			// TODO(lauren) add number type to c1
			continue
		case "custom_static_rich_text":
			// This is just static text, not user input
			continue
		case "nested_field":
			continue
		default:
			l.Info("freshservice-connector: unknown custom field type", zap.Any("serviceItemID", serviceItem.ID), zap.String("type", cf.FieldType))
			continue
		}
		customFields[cfSchema.Id] = cfSchema
	}

	ret := &v2.TicketSchema{
		Id:           schemaID,
		DisplayName:  serviceItem.Name,
		CustomFields: customFields,
		Statuses:     ticketStatuses,
	}
	return ret, nil
}
