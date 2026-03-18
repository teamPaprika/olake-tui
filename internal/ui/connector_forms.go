package ui

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ConnectorType enumerates supported source connector types.
const (
	ConnectorPostgres = "postgres"
	ConnectorMySQL    = "mysql"
	ConnectorMongoDB  = "mongodb"
	ConnectorOracle   = "oracle"
	ConnectorMSSQL    = "mssql"
	ConnectorDB2      = "db2"
	ConnectorKafka    = "kafka"
	ConnectorS3       = "s3"
	// Destination types
	ConnectorIceberg = "iceberg"
	ConnectorParquet = "parquet"
)

// SourceConnectorLabels maps internal type → display label.
var SourceConnectorLabels = []string{
	"PostgreSQL",
	"MySQL",
	"MongoDB",
	"Oracle",
	"MSSQL",
	"DB2",
	"Kafka",
	"S3",
}

// SourceConnectorTypes maps display label index → internal type.
var SourceConnectorTypes = []string{
	ConnectorPostgres,
	ConnectorMySQL,
	ConnectorMongoDB,
	ConnectorOracle,
	ConnectorMSSQL,
	ConnectorDB2,
	ConnectorKafka,
	ConnectorS3,
}

// DestConnectorLabels maps internal type → display label.
var DestConnectorLabels = []string{
	"Apache Iceberg",
	"Amazon S3 (Parquet)",
}

// DestConnectorTypes maps display label index → internal type.
var DestConnectorTypes = []string{
	ConnectorIceberg,
	ConnectorParquet,
}

// ConnectorFieldsForSource returns the form fields for a given source connector type.
func ConnectorFieldsForSource(connectorType string, prefill map[string]string) []FormField {
	get := func(k, def string) string {
		if v, ok := prefill[k]; ok {
			return v
		}
		return def
	}

	switch strings.ToLower(connectorType) {
	case ConnectorPostgres, ConnectorMySQL, ConnectorOracle, ConnectorMSSQL, ConnectorDB2:
		return []FormField{
			{Label: "host", Placeholder: "localhost", Value: get("host", ""), Required: true},
			{Label: "port", Placeholder: "5432", Value: get("port", ""), Required: true},
			{Label: "username", Placeholder: "admin", Value: get("username", ""), Required: true},
			{Label: "password", Placeholder: "••••••••", Value: get("password", ""), Secret: true, Required: true},
			{Label: "database", Placeholder: "mydb", Value: get("database", ""), Required: true},
		}
	case ConnectorMongoDB:
		return []FormField{
			{Label: "connection_string", Placeholder: "mongodb://host:27017", Value: get("connection_string", "")},
			{Label: "host", Placeholder: "localhost (if no connection_string)", Value: get("host", "")},
			{Label: "port", Placeholder: "27017", Value: get("port", "")},
			{Label: "username", Placeholder: "admin", Value: get("username", "")},
			{Label: "password", Placeholder: "••••••••", Value: get("password", ""), Secret: true},
			{Label: "database", Placeholder: "mydb", Value: get("database", "")},
			{Label: "auth_source", Placeholder: "admin", Value: get("auth_source", "")},
			{Label: "replica_set", Placeholder: "rs0", Value: get("replica_set", "")},
		}
	case ConnectorKafka:
		return []FormField{
			{Label: "bootstrap_servers", Placeholder: "localhost:9092", Value: get("bootstrap_servers", ""), Required: true},
			{Label: "topic", Placeholder: "my-topic", Value: get("topic", ""), Required: true},
			{Label: "group_id", Placeholder: "my-group", Value: get("group_id", ""), Required: true},
		}
	case ConnectorS3:
		return []FormField{
			{Label: "bucket", Placeholder: "my-bucket", Value: get("bucket", ""), Required: true},
			{Label: "region", Placeholder: "us-east-1", Value: get("region", ""), Required: true},
			{Label: "access_key", Placeholder: "AKIAIOSFODNN7", Value: get("access_key", ""), Required: true},
			{Label: "secret_key", Placeholder: "••••••••", Value: get("secret_key", ""), Secret: true, Required: true},
			{Label: "prefix", Placeholder: "data/", Value: get("prefix", "")},
		}
	default:
		// Generic fallback
		return []FormField{
			{Label: "host", Placeholder: "localhost", Value: get("host", ""), Required: true},
			{Label: "port", Placeholder: "5432", Value: get("port", ""), Required: true},
			{Label: "username", Placeholder: "admin", Value: get("username", "")},
			{Label: "password", Placeholder: "••••••••", Value: get("password", ""), Secret: true},
			{Label: "database", Placeholder: "mydb", Value: get("database", "")},
		}
	}
}

// ConnectorFieldsForDest returns the form fields for a given destination connector type.
func ConnectorFieldsForDest(connectorType string, prefill map[string]string) []FormField {
	get := func(k, def string) string {
		if v, ok := prefill[k]; ok {
			return v
		}
		return def
	}

	switch strings.ToLower(connectorType) {
	case ConnectorIceberg:
		return []FormField{
			{Label: "catalog_type", Placeholder: "hive / glue / rest", Value: get("catalog_type", ""), Required: true},
			{Label: "warehouse", Placeholder: "s3://my-bucket/warehouse", Value: get("warehouse", ""), Required: true},
			{Label: "uri", Placeholder: "http://catalog:9083", Value: get("uri", ""), Required: true},
			{Label: "credentials", Placeholder: "JSON credentials", Value: get("credentials", "")},
		}
	case ConnectorParquet:
		return []FormField{
			{Label: "storage_type", Placeholder: "s3 / local", Value: get("storage_type", ""), Required: true},
			{Label: "path", Placeholder: "s3://my-bucket/data/ or /local/path", Value: get("path", ""), Required: true},
			{Label: "credentials", Placeholder: "JSON credentials", Value: get("credentials", "")},
		}
	default:
		return []FormField{
			{Label: "path", Placeholder: "/data", Value: get("path", ""), Required: true},
		}
	}
}

// ParseConfigJSON parses a JSON config string into a flat string map.
// Nested structures are ignored; only top-level string/number values are extracted.
func ParseConfigJSON(configJSON string) map[string]string {
	if configJSON == "" {
		return map[string]string{}
	}
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &raw); err != nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		switch vv := v.(type) {
		case string:
			out[k] = vv
		case float64:
			// Format as integer if no fractional part, otherwise keep decimal
			if vv == float64(int64(vv)) {
				out[k] = fmt.Sprintf("%d", int64(vv))
			} else {
				out[k] = fmt.Sprintf("%g", vv)
			}
		}
	}
	return out
}

// BuildConfigJSON serializes a flat field→value map to a JSON config string.
func BuildConfigJSON(values map[string]string) string {
	out := make(map[string]interface{}, len(values))
	for k, v := range values {
		if v != "" {
			out[k] = v
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "{}"
	}
	return string(b)
}
