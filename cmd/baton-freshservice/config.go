package main

import (
	"github.com/conductorone/baton-sdk/pkg/field"
	"github.com/spf13/viper"
)

const apiKey = "api-key"
const domain = "domain"

var (
	apiKeyField         = field.StringField(apiKey, field.WithRequired(true), field.WithDescription("The api key for your account."))
	domainField         = field.StringField(domain, field.WithRequired(true), field.WithDescription("The domain for your account."))
	categoryField       = field.StringField("category-id", field.WithDescription("The category id to filter service items to"))
	configurationFields = []field.SchemaField{apiKeyField, domainField, categoryField}
)

var configRelations = []field.SchemaFieldRelationship{
	field.FieldsDependentOn([]field.SchemaField{categoryField}, []field.SchemaField{field.ListTicketSchemasField}),
}

// ValidateConfig is run after the configuration is loaded, and should return an
// error if it isn't valid. Implementing this function is optional, it only
// needs to perform extra validations that cannot be encoded with configuration
// parameters.
func ValidateConfig(v *viper.Viper) error {
	return nil
}
