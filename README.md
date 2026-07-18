**English** | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) |
[日本語](README.ja.md) | [Español](README.es.md) | [Français](README.fr.md) |
[Deutsch](README.de.md)

# MissionWeaveProtocol Go SDK

<p align="center">
  <img src="https://raw.githubusercontent.com/missionweaveprotocol/missionweaveprotocol/main/assets/brand/missionweaveprotocol-icon.svg" width="160" alt="MissionWeaveProtocol icon">
</p>

<p align="center">
  <strong><a href="https://missionweaveprotocol.github.io/">Official website and documentation</a></strong>
</p>

The MissionWeaveProtocol Go SDK provides schema-first Go bindings for
[MissionWeaveProtocol](https://github.com/missionweaveprotocol/missionweaveprotocol) 0.1. The Go
module is `github.com/missionweaveprotocol/go-sdk`, and its root package is
`missionweaveprotocol`.

This release demonstrates **schema-and-vector conformance**. It does not claim behavioral
conformance for an authoritative Core, Agent runtime, Worker Scheduler, Group gateway, persistence,
or the complete Mission/WorkItem state machine.

## Protocol compatibility

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

SDK and protocol versions are independent. [`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) records the
exact protocol commit plus SHA-256 digests for the vendored schemas and conformance vectors.

## Requirements and installation

Go 1.24 or newer is required.

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## Included capabilities

- byte-exact embedded protocol pin, 21 Draft 2020-12 schemas, and 52 conformance vectors;
- verification of schema, conformance, and combined bundle digests;
- strict UTF-8 JSON parsing with recursive duplicate-member rejection;
- offline `$id` schema resolution with format assertions and ECMAScript-compatible patterns;
- an embedded or caller-supplied `fs.FS` `SchemaCatalog`;
- a 52-vector conformance runner and `missionweaveprotocol-conformance` command;
- RFC 8785 JSON canonicalization and `sha256:` content identifiers;
- Ed25519 signing and verification with unpadded base64url values;
- signing payloads that exclude only the top-level `signature` member;
- `SignedDocumentCodec` coverage for all 22 cryptography cases and 58 evaluations;
- a generic, schema-validating, canonical `FrameCodec` for WebSocket frames.

## Verify the embedded protocol bundle

```go
if err := missionweaveprotocol.VerifyProtocolBundle(); err != nil {
    log.Fatal(err)
}

pin, err := missionweaveprotocol.CurrentProtocolPin()
if err != nil {
    log.Fatal(err)
}
fmt.Println(pin.ProtocolVersion, pin.Commit)
```

## Validate a protocol document

```go
catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
if err != nil {
    log.Fatal(err)
}

if err := catalog.Validate("command.schema.json", commandJSON); err != nil {
    log.Fatal(err)
}
```

`NewSchemaCatalog(source fs.FS)` provides the same Interface for an unpacked protocol checkout or
release bundle. Every schema is registered by `$id` before compilation; unresolved references never
fall back to the network.

## Encode and decode WebSocket frames

```go
codec, err := missionweaveprotocol.NewFrameCodec()
if err != nil {
    log.Fatal(err)
}

frame, err := codec.DecodeFrame(frameJSON)
if err != nil {
    log.Fatal(err)
}

canonicalFrame, err := codec.EncodeFrame(frame)
if err != nil {
    log.Fatal(err)
}
```

`DecodeFrame` rejects malformed UTF-8, duplicate JSON members, unknown frame variants, extra fields,
and schema-invalid content. `EncodeFrame` validates before returning canonical RFC 8785 JSON.

## Canonicalize, hash, and sign

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`, `CanonicalHash`, and the document-signing Interface accept JSON bytes and do
not apply custom conversions for Go values such as `time.Time`. `MarshalCanonicalJSON` is an
explicit convenience that applies standard `encoding/json` marshaling before JCS. `SignDocument`
and `VerifyDocument` remove the top-level `signature` member before canonicalization; nested
members with that name remain signed.

## Sign and verify Signed Documents

`SignedDocumentCodec` implements the ordered cryptographic profile for exactly nine explicit
document kinds:

```go
codec, err := missionweaveprotocol.NewSignedDocumentCodec()
signed, err := codec.Sign(missionweaveprotocol.SignedDocumentCommand, unsigned, signingKey)
verified, err := codec.Verify(missionweaveprotocol.SignedDocumentCommand, raw, keyResolver)
fmt.Println(signed["signature"], verified.SigningHash(), verified.ResolvedKey().Principal())
```

`SigningKey` is the only signing adapter. `KeyResolver` receives a `KeyResolutionRequest` and must
return a `KeyRegistrySnapshot` whose completeness is explicitly
`KeyRegistryOrganizationWide`; partial or unspecified Agent Registry snapshots fail closed. Verification
errors expose only a stable `WireCode()` to peers, while `ProtectedDiagnostic()` retains the first
failing stage and reason for local operators. See the runnable test-fixture example in
[`examples/sign`](examples/sign).

## Run conformance

Run against the embedded protocol bundle:

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

Or run against a protocol checkout or release bundle:

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

Success reports `52/52 conformance vectors passed`. The command exits non-zero for a validity
mismatch, malformed vector, missing resource, or schema compilation error.

## Examples and development

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

The CI gate also verifies formatting, canonical naming, both embedded and checkout conformance,
and a compiled-binary resource smoke test.

## Scope

The normative protocol repository remains the source of truth. This SDK intentionally does not
copy the Python reference implementation's server, database adapters, scheduling algorithm, local
runtime, or internal projection models. Future runtime features require their own behavioral
conformance work and will be documented separately.

## License

Licensed under [Apache-2.0](LICENSE).
