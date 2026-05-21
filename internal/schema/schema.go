package schema

import (
    _ "embed"
)

//go:embed schema.json
var schemaJSON []byte

func Contents() ([]byte, error) {
    return schemaJSON, nil
}
