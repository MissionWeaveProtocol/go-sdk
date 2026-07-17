# MissionWeaveProtocol Go SDK

Official Go SDK for
[MissionWeaveProtocol](https://github.com/missionweaveprotocol/missionweaveprotocol).

This repository currently provides the package and verification foundation. Protocol artifacts and
bindings are added in subsequent, independently reviewed changes.

## Development

Go 1.24 or newer is required.

```bash
go test ./...
go vet ./...
go build ./...
go run ./internal/cmd/repository-policy
```

## License

Licensed under [Apache-2.0](LICENSE).
