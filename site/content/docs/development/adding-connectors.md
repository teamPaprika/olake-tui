---
title: "Adding Connectors"
weight: 3
---

# Adding Connectors

This guide explains how to add a new source or destination connector to olake-tui. All connector-specific UI logic lives in `internal/ui/connector_forms/`.

## Overview

Each connector needs:

1. **Field definitions** — What configuration fields to display
2. **Validation** — Rules for required fields, formats, and ranges
3. **JSON conversion** — How form data maps to the connector's config JSON

## Step 1: Define Fields

Open `connector_forms.go` and add a new field definition function. Each field is defined with a type, label, placeholder, and validation rules.

```go
func MyConnectorFields() []FormField {
    return []FormField{
        {
            Key:         "host",
            Label:       "Host",
            Placeholder: "localhost",
            Required:    true,
            Type:        FieldText,
        },
        {
            Key:         "port",
            Label:       "Port",
            Placeholder: "5432",
            Required:    true,
            Type:        FieldNumber,
            Validation:  ValidatePort,
        },
        {
            Key:         "database",
            Label:       "Database",
            Placeholder: "mydb",
            Required:    true,
            Type:        FieldText,
        },
        {
            Key:         "username",
            Label:       "Username",
            Required:    true,
            Type:        FieldText,
        },
        {
            Key:         "password",
            Label:       "Password",
            Required:    true,
            Type:        FieldPassword,
        },
        {
            Key:         "ssl_mode",
            Label:       "SSL Mode",
            Type:        FieldSelect,
            Options:     []string{"disable", "require", "verify-ca", "verify-full"},
            Default:     "disable",
        },
    }
}
```

### Field Types

| Type | Rendered As | Use For |
|------|------------|---------|
| `FieldText` | Text input | Hostnames, database names, paths |
| `FieldPassword` | Masked input | Passwords, API keys, secrets |
| `FieldNumber` | Numeric input | Ports, timeouts, batch sizes |
| `FieldSelect` | Dropdown/list | Enums, modes, SSL options |
| `FieldToggle` | Checkbox | Boolean flags (enable/disable) |
| `FieldTextArea` | Multi-line input | JSON configs, queries |

## Step 2: Register the Connector

Add your connector to the registry map so the UI knows how to render it:

```go
// In connector_forms.go
var ConnectorFieldRegistry = map[string]func() []FormField{
    "mongodb":    MongoDBFields,
    "mysql":      MySQLFields,
    "postgres":   PostgreSQLFields,
    "s3":         S3Fields,
    "my_connector": MyConnectorFields,  // Add this line
}
```

For sources, also add it to the source type list. For destinations, add it to the destination type list.

## Step 3: Validation

Each field can have a validation function. Common validators are provided:

```go
// Built-in validators
ValidateRequired    // Non-empty
ValidatePort        // 1-65535
ValidateURL         // Valid URL format
ValidateJSON        // Valid JSON string

// Custom validator example
func ValidateMyField(value string) error {
    if !strings.HasPrefix(value, "my-") {
        return fmt.Errorf("must start with 'my-'")
    }
    return nil
}
```

Assign validators to fields:

```go
{
    Key:        "endpoint",
    Label:      "Endpoint URL",
    Required:   true,
    Type:       FieldText,
    Validation: ValidateURL,
}
```

## Step 4: JSON Conversion

Form values are collected as `map[string]string` and need to be converted to the connector's expected JSON config format. The conversion happens automatically for simple key-value mappings. For nested structures, add a custom converter:

```go
func MyConnectorToJSON(fields map[string]string) (map[string]interface{}, error) {
    port, err := strconv.Atoi(fields["port"])
    if err != nil {
        return nil, fmt.Errorf("invalid port: %w", err)
    }

    return map[string]interface{}{
        "host":     fields["host"],
        "port":     port,
        "database": fields["database"],
        "username": fields["username"],
        "password": fields["password"],
        "ssl_mode": fields["ssl_mode"],
    }, nil
}
```

The resulting JSON is encrypted with AES-256-GCM before storage.

## Step 5: Test

Add unit tests for your connector's field definitions and JSON conversion:

```bash
go test ./internal/ui/connector_forms/... -run TestMyConnector
```

Verify:
- All required fields are marked correctly
- Validation rejects invalid input
- JSON output matches the format expected by OLake connectors
- Round-trip: form → JSON → encrypt → decrypt → form works correctly

## Reference

For the full list of supported OLake connectors and their configuration formats, see the [OLake Documentation](https://olake.io/docs/).

Existing connector implementations in `connector_forms.go` are the best reference for patterns and conventions.
