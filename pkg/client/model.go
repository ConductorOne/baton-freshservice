package client

import "time"

const ServiceItemVisibilityDraft = 1

type auth struct {
	bearerToken string
}

type AgentsAPIData struct {
	Agents []Agent `json:"agents,omitempty"`
}

type Agent struct {
	Active      bool        `json:"active,omitempty"`
	Address     string      `json:"address,omitempty"`
	Email       string      `json:"email,omitempty"`
	FirstName   string      `json:"first_name,omitempty"`
	ID          int64       `json:"id,omitempty"`
	LastName    string      `json:"last_name,omitempty"`
	Roles       []AgentRole `json:"roles,omitempty"`
	LastLoginAt time.Time   `json:"last_login_at,omitempty"`
}

type AgentDetailAPIData struct {
	Agent Agent `json:"agent,omitempty"`
}

type AgentRole struct {
	RoleID          int64   `json:"role_id,omitempty"`
	AssignmentScope string  `json:"assignment_scope,omitempty"`
	Groups          []int64 `json:"groups,omitempty"`
}

type RolesAPIData struct {
	Roles []Roles `json:"roles,omitempty"`
}

type Roles struct {
	Description string   `json:"description,omitempty"`
	Privileges  []string `json:"privileges,omitempty"`
	ID          int64    `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	RoleType    int      `json:"role_type,omitempty"`
}

type AgentGroupsAPIData struct {
	Groups []AgentGroup `json:"groups,omitempty"`
}

type AgentGroup struct {
	ID          int64   `json:"id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Members     []int64 `json:"members"`
}

type AgentGroupDetailAPIData struct {
	Group AgentGroup `json:"group,omitempty"`
}

type UpdateAgentRoles struct {
	Roles []AgentRole `json:"roles"`
}

type requestersAPIData struct {
	Requesters []Requesters `json:"requesters,omitempty"`
}

type Requesters struct {
	Active       bool   `json:"active,omitempty"`
	Address      string `json:"address,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	ID           int64  `json:"id,omitempty"`
	IsAgent      bool   `json:"is_agent,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	PrimaryEmail string `json:"primary_email,omitempty"`
}

type RequesterGroupsAPIData struct {
	RequesterGroups []RequesterGroup `json:"requester_groups,omitempty"`
}

type RequesterGroup struct {
	ID          int64  `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

type requesterGroupMembersAPIData struct {
	Requesters []RequesterGroupMember `json:"requesters,omitempty"`
}

type RequesterGroupMember struct {
	ID        int    `json:"id,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
}

// Ticket models.
type TicketUpdatePayload struct {
	Description string   `json:"description,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type TicketFieldsResponse struct {
	TicketFields []TicketField `json:"ticket_fields"`
}

type TicketField struct {
	ID                 int           `json:"id"`
	WorkspaceID        *int          `json:"workspace_id,omitempty"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
	Name               string        `json:"name"`
	Label              string        `json:"label"`
	Description        string        `json:"description"`
	FieldType          string        `json:"field_type"`
	Required           bool          `json:"required"`
	RequiredForClosure bool          `json:"required_for_closure"`
	DefaultField       bool          `json:"default_field"`
	Choices            []Choice      `json:"choices"`
	NestedFields       []NestedField `json:"nested_fields"`
	Sections           []Section     `json:"sections"`
	BelongsToSection   bool          `json:"belongs_to_section"`
}

type Choice struct {
	ID            int      `json:"id"`
	Value         string   `json:"value"`
	DisplayID     *int     `json:"display_id,omitempty"`
	NestedOptions []Choice `json:"nested_options,omitempty"`
}

type NestedField struct {
	Name          string    `json:"name"`
	ID            int       `json:"id"`
	Label         string    `json:"label"`
	LabelInPortal string    `json:"label_in_portal"`
	Level         int       `json:"level"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	FieldID       int       `json:"field_id"`
}

type Section struct {
	// Define the fields of a section if necessary
}

type TicketResponse struct {
	Ticket *TicketDetails `json:"ticket"`
}

type TicketDetails struct {
	ID                 int         `json:"id"`
	Type               string      `json:"type"`
	RequesterID        int         `json:"requester_id"`
	RequestedForID     int         `json:"requested_for_id"`
	ResponderID        *int        `json:"responder_id"`
	Source             int         `json:"source"`
	Status             int         `json:"status"`
	Subject            string      `json:"subject"`
	WorkspaceID        int         `json:"workspace_id"`
	Description        string      `json:"description"`
	DescriptionText    string      `json:"description_text"`
	CustomFields       interface{} `json:"custom_fields"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
	Category           *string     `json:"category"`
	SubCategory        *string     `json:"sub_category"`
	ItemCategory       *string     `json:"item_category"`
	Deleted            bool        `json:"deleted"`
	ApprovalStatus     int         `json:"approval_status"`
	ApprovalStatusName string      `json:"approval_status_name"`
	Tags               []string    `json:"tags"`
}

type CustomFieldOption func(serviceRequestPayload *ServiceRequestPayload)

func WithCustomField(id string, value interface{}) CustomFieldOption {
	return func(serviceRequestPayload *ServiceRequestPayload) {
		if serviceRequestPayload.CustomFields == nil {
			serviceRequestPayload.CustomFields = make(map[string]interface{})
		}
		serviceRequestPayload.CustomFields[id] = value
	}
}

type ServiceRequestPayload struct {
	RequestedFor string                 `json:"requested_for,omitempty"`
	Email        string                 `json:"email"`
	Quantity     int                    `json:"quantity"`
	CustomFields map[string]interface{} `json:"custom_fields"`
}

type ServiceRequest struct {
	ApprovalStatusName string `json:"approval_status_name"`
	ApprovalStatus     string `json:"approval_status"`
	TicketDetails
}

type ServiceRequestResponse struct {
	ServiceRequest *ServiceRequest `json:"service_request"`
}

type ServiceCatalogItemResponse struct {
	ServiceItem *ServiceItem `json:"service_item"`
}

type ServiceCatalogItemsListResponse struct {
	ServiceItems []*ServiceItem `json:"service_items"`
}

type ServiceItem struct {
	ID           int64         `json:"id"`
	Name         string        `json:"name"`
	DisplayID    int64         `json:"display_id"`
	CustomFields []CustomField `json:"custom_fields"`
	CategoryID   int64         `json:"category_id"`
	ProductID    *int64        `json:"product_id"`
	Deleted      bool          `json:"deleted"`
	ItemType     int           `json:"item_type"`
	CITypeID     *int64        `json:"ci_type_id"`
	Visibility   int           `json:"visibility"` // 1 denotes draft and 2 denotes published.
	WorkspaceID  int           `json:"workspace_id"`
	IsBundle     bool          `json:"is_bundle"`
	CreateChild  bool          `json:"create_child"`
	ChildItems   []interface{} `json:"child_items"`
}

type CustomField struct {
	ID           string        `json:"id"`
	Label        string        `json:"label"`
	Name         string        `json:"name"`
	FieldType    string        `json:"field_type"`
	ItemID       int64         `json:"item_id"`
	Required     bool          `json:"required"`
	Choices      [][]string    `json:"choices"`
	FieldOptions FieldOptions  `json:"field_options"`
	NestedFields []interface{} `json:"nested_fields"`
	Deleted      bool          `json:"deleted"`
}

type FieldOptions struct {
	VisibleInAgentPortal string `json:"visible_in_agent_portal"`
	PDF                  string `json:"pdf"`
	IMAPIName            string `json:"im_api_name"`
	RequiredForCreate    string `json:"required_for_create"`
	RequesterCanEdit     string `json:"requester_can_edit"`
	VisibleInPublic      string `json:"visible_in_public"`
	RequiredForClosure   string `json:"required_for_closure"`
	DisplayedToRequester string `json:"displayed_to_requester"`
	Placeholder          string `json:"placeholder"`

	// Used for dynamic dropdown
	Link        string `json:"link"`
	DataSource  string `json:"data_source"`
	Conditions  string `json:"conditions"`
	SameAsAgent string `json:"same_as_agent"`
}
