[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) |
[日本語](README.ja.md) | [Español](README.es.md) | [Français](README.fr.md) |
**Deutsch**

# MissionWeaveProtocol Go SDK

<p align="center">
  <img src="https://raw.githubusercontent.com/missionweaveprotocol/missionweaveprotocol/main/assets/brand/missionweaveprotocol-icon.svg" width="160" alt="MissionWeaveProtocol-Symbol">
</p>

<p align="center">
  <strong><a href="https://missionweaveprotocol.github.io/">Offizielle Website und Dokumentation</a></strong>
</p>

Das MissionWeaveProtocol Go SDK stellt schema-first Go bindings für
[MissionWeaveProtocol](https://github.com/missionweaveprotocol/missionweaveprotocol) 0.1 bereit. Das
Go module heißt `github.com/missionweaveprotocol/go-sdk`, das root package
`missionweaveprotocol`.

Diese Version weist ausschließlich **Schema- und Vektorkonformität** nach. Sie beansprucht keine
Verhaltenskonformität für einen autoritativen Core, Agent Runtime, Worker Scheduler, Group Gateway,
Persistenz oder die vollständige Mission/WorkItem-Zustandsmaschine.

## Protokollkompatibilität

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

SDK und Protokoll werden unabhängig versioniert. [`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) hält den
exakten Protokoll-commit und die SHA-256 digest der vendored schemas und conformance vectors fest.

## Voraussetzungen und Installation

Go 1.24 oder neuer ist erforderlich.

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## Enthaltene Fähigkeiten

- byte-exakter embedded protocol pin, 21 Draft-2020-12-schemas und 43 conformance vectors;
- Prüfung der schema-, conformance- und kombinierten bundle digest;
- striktes UTF-8-JSON-Parsing mit rekursiver Ablehnung doppelter members;
- offline `$id`-schema-Auflösung mit format assertions und ECMAScript-kompatiblen patterns;
- `SchemaCatalog` über das embedded oder vom Aufrufer bereitgestellte `fs.FS`;
- 43-vector conformance runner und der Befehl `missionweaveprotocol-conformance`;
- RFC-8785-JSON-canonicalization und `sha256:` content identifiers;
- Ed25519-Signieren und -Prüfen mit base64url ohne padding;
- Signatur-payloads, die ausschließlich das oberste `signature` member ausschließen;
- ein generic, schema-validating und canonical `FrameCodec` für WebSocket frames.

## Embedded protocol bundle prüfen

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

## Protokolldokument validieren

```go
catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
if err != nil {
    log.Fatal(err)
}

if err := catalog.Validate("command.schema.json", commandJSON); err != nil {
    log.Fatal(err)
}
```

`NewSchemaCatalog(source fs.FS)` stellt dieselbe Interface für einen entpackten protocol checkout
oder release bundle bereit. Alle schemas werden vor dem Kompilieren über `$id` registriert;
unaufgelöste Referenzen greifen niemals auf das Netzwerk zurück.

## WebSocket frames codieren und decodieren

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

`DecodeFrame` lehnt fehlerhaftes UTF-8, doppelte JSON members, unbekannte frame variants,
zusätzliche Felder und schema-invalid content ab. `EncodeFrame` validiert vor der Ausgabe von
canonical RFC-8785-JSON.

## Canonicalize, Hash berechnen und signieren

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`, `CanonicalHash` und die Document-Signing Interface akzeptieren JSON bytes und
wenden keine eigenen Konvertierungen auf Go-Werte wie `time.Time` an. `MarshalCanonicalJSON` ist
eine explizite Hilfsfunktion, die vor JCS das standardmäßige `encoding/json` marshaling ausführt.
`SignDocument` und `VerifyDocument` entfernen vor der canonicalization das oberste `signature`
member; verschachtelte members gleichen Namens bleiben Teil der Signatur.

## Conformance ausführen

Gegen das embedded protocol bundle:

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

Gegen einen protocol checkout oder release bundle:

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

Bei Erfolg wird `43/43 conformance vectors passed` ausgegeben. Der Befehl endet bei validity
mismatch, malformed vector, fehlender Ressource oder schema compilation error mit einem Status
ungleich null.

## Beispiele und Entwicklung

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

Das CI gate prüft außerdem formatting, canonical naming, embedded und checkout conformance sowie
einen compiled binary resource smoke test.

## Umfang

Das normative Protokoll-repository bleibt die source of truth. Dieses SDK kopiert bewusst weder
server noch database adapters, scheduling algorithm, local runtime oder internal projection models
der Python-Referenzimplementierung. Künftige runtime-Funktionen benötigen eigenständige behavioral
conformance-Arbeit und werden separat dokumentiert.

## Lizenz

Lizenziert unter [Apache-2.0](LICENSE).
