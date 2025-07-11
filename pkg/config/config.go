package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

const apiKey = "api-key"
const domain = "domain"

var (
	apiKeyField = field.StringField(
		apiKey,
		field.WithRequired(true),
		field.WithIsSecret(true),
		field.WithDisplayName("API key"),
		field.WithDescription("The api key for your account."),
	)
	domainField = field.StringField(
		domain,
		field.WithRequired(true),
		field.WithDisplayName("Domain"),
		field.WithDescription("The domain for your account."),
	)
	categoryField = field.StringField(
		"category-id",
		field.WithDisplayName("Category ID"),
		field.WithDescription("The category id to filter service items to"),
	)
	externalTicketField = field.TicketingField.ExportAs(field.ExportTargetGUI)
	configurationFields = []field.SchemaField{apiKeyField, domainField, categoryField, externalTicketField}
)

var configRelations = []field.SchemaFieldRelationship{
	field.FieldsDependentOn([]field.SchemaField{categoryField}, []field.SchemaField{field.TicketingField}),
}

//go:generate go run ./gen
var Config = field.NewConfiguration(
	configurationFields,
	field.WithConstraints(configRelations...),
	field.WithConnectorDisplayName("Freshservice"),
	field.WithHelpUrl("/docs/baton/freshservice"),
	field.WithIconUrl("/static/app-icons/freshservice.svg"),
)
