package client

import (
	"time"
)

type auth struct {
	bearerToken string
}

type AgentsAPIDataV2 struct {
	Agents []Agents `json:"agents,omitempty"`
}

type Agents struct {
	Active                                    bool                 `json:"active,omitempty"`
	Address                                   interface{}          `json:"address,omitempty"`
	APIKeyEnabled                             bool                 `json:"api_key_enabled,omitempty"`
	AutoAssignStatusChangedAt                 interface{}          `json:"auto_assign_status_changed_at,omitempty"`
	AutoAssignTickets                         bool                 `json:"auto_assign_tickets,omitempty"`
	BackgroundInformation                     interface{}          `json:"background_information,omitempty"`
	CanSeeAllTicketsFromAssociatedDepartments bool                 `json:"can_see_all_tickets_from_associated_departments,omitempty"`
	CreatedAt                                 time.Time            `json:"created_at,omitempty"`
	CustomFields                              AgentCustomFields    `json:"custom_fields,omitempty"`
	DepartmentIDs                             []interface{}        `json:"department_ids,omitempty"`
	DepartmentNames                           interface{}          `json:"department_names,omitempty"`
	Email                                     string               `json:"email,omitempty"`
	ExternalID                                interface{}          `json:"external_id,omitempty"`
	FirstName                                 string               `json:"first_name,omitempty"`
	HasLoggedIn                               bool                 `json:"has_logged_in,omitempty"`
	ID                                        int64                `json:"id,omitempty"`
	JobTitle                                  interface{}          `json:"job_title,omitempty"`
	Language                                  string               `json:"language,omitempty"`
	LastActiveAt                              time.Time            `json:"last_active_at,omitempty"`
	LastLoginAt                               time.Time            `json:"last_login_at,omitempty"`
	LastName                                  string               `json:"last_name,omitempty"`
	LocationID                                interface{}          `json:"location_id,omitempty"`
	LocationName                              interface{}          `json:"location_name,omitempty"`
	MobilePhoneNumber                         interface{}          `json:"mobile_phone_number,omitempty"`
	Occasional                                bool                 `json:"occasional,omitempty"`
	ReportingManagerID                        interface{}          `json:"reporting_manager_id,omitempty"`
	Roles                                     []AgentRoles         `json:"roles,omitempty"`
	ScoreboardLevelID                         interface{}          `json:"scoreboard_level_id,omitempty"`
	ScoreboardPoints                          interface{}          `json:"scoreboard_points,omitempty"`
	Signature                                 interface{}          `json:"signature,omitempty"`
	TimeFormat                                string               `json:"time_format,omitempty"`
	TimeZone                                  string               `json:"time_zone,omitempty"`
	UpdatedAt                                 time.Time            `json:"updated_at,omitempty"`
	VipUser                                   bool                 `json:"vip_user,omitempty"`
	WorkPhoneNumber                           interface{}          `json:"work_phone_number,omitempty"`
	WorkScheduleID                            int64                `json:"work_schedule_id,omitempty"`
	WorkspaceIDs                              []int                `json:"workspace_ids,omitempty"`
	WorkspaceInfo                             []AgentWorkspaceInfo `json:"workspace_info,omitempty"`
	MemberOf                                  []interface{}        `json:"member_of,omitempty"`
	ObserverOf                                []interface{}        `json:"observer_of,omitempty"`
	MemberOfPendingApproval                   []interface{}        `json:"member_of_pending_approval,omitempty"`
	ObserverOfPendingApproval                 []interface{}        `json:"observer_of_pending_approval,omitempty"`
}

type AgentDetailAPIData struct {
	Agent Agents `json:"agent"`
}

type AgentRoles struct {
	RoleID          int64         `json:"role_id,omitempty"`
	AssignmentScope string        `json:"assignment_scope,omitempty"`
	Groups          []interface{} `json:"groups,omitempty"`
}

type AgentWorkspaceInfo struct {
	WorkspaceID       int         `json:"workspace_id,omitempty"`
	ScoreboardLevelID interface{} `json:"scoreboard_level_id,omitempty"`
	Points            interface{} `json:"points,omitempty"`
}

type AgentCustomFields struct {
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

type RolesAPIDataV2 struct {
	Roles []Roles `json:"roles,omitempty"`
	Meta  Meta    `json:"meta,omitempty"`
}

type Roles struct {
	Description string    `json:"description,omitempty"`
	Privileges  []string  `json:"privileges,omitempty"`
	ID          int64     `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	Default     bool      `json:"default,omitempty"`
	RoleType    int       `json:"role_type,omitempty"`
}

type Meta struct {
	UsedCoPilotCount int `json:"used_co_pilot_count"`
	CoPilotLicCount  int `json:"co_pilot_lic_count"`
}

type GroupContact struct {
	Name   string   `json:"name,omitempty"`
	Avatar struct{} `json:"avatar,omitempty"`
	Email  string   `json:"email,omitempty"`
}

type GroupsAPIDataV2 struct {
	Groups []Group `json:"groups,omitempty"`
}

type Groups struct {
	ID                       int64         `json:"id,omitempty"`
	Name                     string        `json:"name,omitempty"`
	Description              string        `json:"description,omitempty"`
	EscalateTo               interface{}   `json:"escalate_to,omitempty"`
	UnassignedFor            interface{}   `json:"unassigned_for,omitempty"`
	BusinessHoursID          interface{}   `json:"business_hours_id,omitempty"`
	CreatedAt                time.Time     `json:"created_at,omitempty"`
	UpdatedAt                time.Time     `json:"updated_at,omitempty"`
	AutoTicketAssign         bool          `json:"auto_ticket_assign,omitempty"`
	Restricted               bool          `json:"restricted,omitempty"`
	ApprovalRequired         bool          `json:"approval_required,omitempty"`
	OcsScheduleID            interface{}   `json:"ocs_schedule_id,omitempty"`
	WorkspaceID              int           `json:"workspace_id,omitempty"`
	Members                  []int64       `json:"members,omitempty"`
	Observers                []interface{} `json:"observers,omitempty"`
	Leaders                  []interface{} `json:"leaders,omitempty"`
	MembersPendingApproval   []interface{} `json:"members_pending_approval,omitempty"`
	LeadersPendingApproval   []interface{} `json:"leaders_pending_approval,omitempty"`
	ObserversPendingApproval []interface{} `json:"observers_pending_approval,omitempty"`
}

type GroupDetailAPIData struct {
	Group Groups `json:"group"`
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

type UpdateAgentRoles struct {
	RoleIDs []int64 `json:"role_ids"`
}

type RemoveAgentFromGroup struct {
	Agents []struct {
		ID      int  `json:"id"`
		Deleted bool `json:"deleted"`
	} `json:"agents"`
}

type AddAgentToGroup struct {
	Agents []struct {
		ID int `json:"id"`
	} `json:"agents"`
}
