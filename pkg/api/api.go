// Package api provides access to the OpenAPI specification.
//
//nolint:revive // standard package name
package api

import _ "embed"

// OpenAPISpec contains the raw bytes of the OpenAPI YAML file.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
