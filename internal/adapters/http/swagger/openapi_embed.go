package swagger

import _ "embed"

// OpenAPI contains the embedded OpenAPI YAML specification.
//
//go:embed openapi.yaml
var OpenAPI []byte

// RedocJS contains the embedded ReDoc standalone JavaScript.
//
//go:embed static/redoc.standalone.js
var RedocJS []byte
