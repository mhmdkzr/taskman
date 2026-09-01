package migrations

import _ "embed"

//go:embed schema.sql
var schemaSQL string

func GetSchemaSQL() string {
	return schemaSQL
}
