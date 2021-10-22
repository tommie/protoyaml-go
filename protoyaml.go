// Package protoyaml contains a YAML decoder in the spirit of what
// https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson is
// for JSON.
//
// Special Considerations
//
// * YAML names correspond to Protobuf names, not JSON-names.
// * Enums can be provied as names or numbers.
package protoyaml
