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
