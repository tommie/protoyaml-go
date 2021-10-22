//go:generate sh -c "cd ../.. && protoc --plugin=\"$(go env GOPATH)/bin/protoc-gen-go\" --go_out=. --go_opt=paths=source_relative internal/testproto/*.proto"
package testproto
