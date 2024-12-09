package client

type auth struct {
	bearerToken string
}

type AgentsAPIData struct {
	Agents []Agents `json:"agents,omitempty"`
}

type Agents struct {
	Active    bool         `json:"active,omitempty"`
	Address   string       `json:"address,omitempty"`
	Email     string       `json:"email,omitempty"`
	FirstName string       `json:"first_name,omitempty"`
	ID        int64        `json:"id,omitempty"`
	LastName  string       `json:"last_name,omitempty"`
	Roles     []AgentRoles `json:"roles,omitempty"`
}

type AgentDetailAPIData struct {
	Agent Agents `json:"agent,omitempty"`
}

type AgentRoles struct {
	RoleID          int64    `json:"role_id,omitempty"`
	AssignmentScope string   `json:"assignment_scope,omitempty"`
	Groups          []string `json:"groups,omitempty"`
}

type Group struct {
	ID          int64   `json:"id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	AgentIDs    []int64 `json:"agent_ids,omitempty"`
	Type        string  `json:"type,omitempty"`
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

type GroupsAPIData struct {
	Groups []Group `json:"groups,omitempty"`
}

type Groups struct {
	ID          int64   `json:"id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Members     []int64 `json:"members,omitempty"`
}

type GroupDetailAPIData struct {
	Group Groups `json:"group,omitempty"`
}

type UpdateAgentRoles struct {
	Roles []BodyRole `json:"roles"`
}

type BodyRole struct {
	RoleID          int64  `json:"role_id"`
	AssignmentScope string `json:"assignment_scope"`
}

type GroupMembers struct {
	Members []int64 `json:"members"`
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

type requesterAPIData struct {
	Requesters []Requester `json:"requesters,omitempty"`
}

type Requester struct {
	ID        int    `json:"id,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
}
