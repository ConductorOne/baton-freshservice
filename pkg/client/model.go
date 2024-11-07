package client

import (
	"time"
)

type auth struct {
	bearerToken string
}

type AgentsAPIData []Agent

type Agent struct {
	Available      bool      `json:"available,omitempty"`
	Occasional     bool      `json:"occasional,omitempty"`
	ID             int64     `json:"id,omitempty"`
	TicketScope    int       `json:"ticket_scope,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	LastActiveAt   time.Time `json:"last_active_at,omitempty"`
	AvailableSince any       `json:"available_since,omitempty"`
	Type           string    `json:"type,omitempty"`
	Contact        Contact   `json:"contact,omitempty"`
	Deactivated    bool      `json:"deactivated,omitempty"`
	Signature      any       `json:"signature,omitempty"`
	FocusMode      bool      `json:"focus_mode,omitempty"`
}

type Contact struct {
	Active      bool      `json:"active,omitempty"`
	Email       string    `json:"email,omitempty"`
	JobTitle    any       `json:"job_title,omitempty"`
	Language    string    `json:"language,omitempty"`
	LastLoginAt any       `json:"last_login_at"`
	Mobile      any       `json:"mobile,omitempty"`
	Name        string    `json:"name,omitempty"`
	Phone       any       `json:"phone,omitempty"`
	TimeZone    string    `json:"time_zone,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type GroupsAPIData []Group

type Group struct {
	ID                              int64                    `json:"id,omitempty"`
	Name                            string                   `json:"name,omitempty"`
	Description                     interface{}              `json:"description,omitempty"`
	EscalateTo                      interface{}              `json:"escalate_to,omitempty"`
	UnassignedFor                   interface{}              `json:"unassigned_for,omitempty"`
	AgentIDs                        []int64                  `json:"agent_ids,omitempty"`
	CreatedAt                       time.Time                `json:"created_at,omitempty"`
	UpdatedAt                       time.Time                `json:"updated_at,omitempty"`
	AllowAgentsToChangeAvailability bool                     `json:"allow_agents_to_change_availability,omitempty"`
	AgentAvailabilityStatus         bool                     `json:"agent_availability_status,omitempty"`
	BusinessCalendarID              int64                    `json:"business_calendar_id,omitempty"`
	Type                            string                   `json:"type,omitempty"`
	AutomaticAgentAssignment        AutomaticAgentAssignment `json:"automatic_agent_assignment,omitempty"`
}

type AutomaticAgentAssignment struct {
	Enabled bool `json:"enabled,omitempty"`
}

type RolesAPIData []Role
type Role struct {
	ID          int64     `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Default     bool      `json:"default,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	AgentType   int       `json:"agent_type,omitempty"`
}

type GroupRolesAPIData []GroupRoles

type GroupContact struct {
	Name   string   `json:"name,omitempty"`
	Avatar struct{} `json:"avatar,omitempty"`
	Email  string   `json:"email,omitempty"`
}

type GroupRoles struct {
	ID          int64        `json:"id,omitempty"`
	TicketScope int          `json:"ticket_scope,omitempty"`
	WriteAccess bool         `json:"write_access,omitempty"`
	RoleIDs     []int64      `json:"role_ids,omitempty"`
	Contact     GroupContact `json:"contact,omitempty"`
	CreatedAt   time.Time    `json:"created_at,omitempty"`
	UpdatedAt   time.Time    `json:"updated_at,omitempty"`
	Deactivated bool         `json:"deactivated,omitempty"`
}

type AgentDetailsAPIData struct {
	ID                      int64          `json:"id,omitempty"`
	OrgAgentID              string         `json:"org_agent_id,omitempty"`
	GroupIDs                []int64        `json:"group_ids,omitempty"`
	OrgGroupIDs             []any          `json:"org_group_ids,omitempty"`
	Deactivated             bool           `json:"deactivated,omitempty"`
	Available               bool           `json:"available,omitempty"`
	AvailableSince          any            `json:"available_since,omitempty"`
	Occasional              bool           `json:"occasional,omitempty"`
	TicketScope             int            `json:"ticket_scope,omitempty"`
	RoleIDs                 []int64        `json:"role_ids,omitempty"`
	SkillIDs                []any          `json:"skill_ids,omitempty"`
	Type                    string         `json:"type,omitempty"`
	LastActiveAt            time.Time      `json:"last_active_at,omitempty"`
	CreatedAt               time.Time      `json:"created_at,omitempty"`
	UpdatedAt               time.Time      `json:"updated_at,omitempty"`
	ContributionGroupIDs    []any          `json:"contribution_group_ids,omitempty"`
	OrgContributionGroupIDs []any          `json:"org_contribution_group_ids,omitempty"`
	Signature               any            `json:"signature,omitempty"`
	Contact                 Contact        `json:"contact,omitempty"`
	Scope                   int            `json:"scope,omitempty"`
	Availability            []Availability `json:"availability,omitempty"`
	FocusMode               bool           `json:"focus_mode,omitempty"`
}

type Availability struct {
	Channel        string    `json:"channel,omitempty"`
	Available      bool      `json:"available,omitempty"`
	AvailableSince time.Time `json:"available_since,omitempty"`
}
