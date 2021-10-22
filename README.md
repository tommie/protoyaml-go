# ProtoYAML

A [YAML](https://yaml.org/) decoder for [Go](https://golang.org/) in
the spirit of what
[protojson](https://pkg.go.dev/google.golang.org/protobuf/encoding/protojson)
is for JSON.

## Maturity

This is in early development.

## Special Considerations

* YAML names correspond to Protobuf names, not JSON-names.
* Enums can be provied as names or numbers.

## Running Tests

```shell
$ go generate ./..
$ go test ./...
```

## License

[MIT License](./LICENSE)
