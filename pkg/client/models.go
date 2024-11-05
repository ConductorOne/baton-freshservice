package client

import "time"

type auth struct {
	bearerToken string
}

type AgentsAPIData []struct {
	Available      bool      `json:"available"`
	Occasional     bool      `json:"occasional"`
	ID             int64     `json:"id"`
	TicketScope    int       `json:"ticket_scope"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastActiveAt   time.Time `json:"last_active_at"`
	AvailableSince any       `json:"available_since"`
	Type           string    `json:"type"`
	Contact        struct {
		Active      bool      `json:"active"`
		Email       string    `json:"email"`
		JobTitle    any       `json:"job_title"`
		Language    string    `json:"language"`
		LastLoginAt any       `json:"last_login_at"`
		Mobile      any       `json:"mobile"`
		Name        string    `json:"name"`
		Phone       any       `json:"phone"`
		TimeZone    string    `json:"time_zone"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	} `json:"contact"`
	Deactivated bool `json:"deactivated"`
	Signature   any  `json:"signature"`
	FocusMode   bool `json:"focus_mode"`
}
