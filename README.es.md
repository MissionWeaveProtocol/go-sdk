[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) |
[日本語](README.ja.md) | **Español** | [Français](README.fr.md) |
[Deutsch](README.de.md)

# MissionWeaveProtocol Go SDK

<p align="center">
  <img src="https://raw.githubusercontent.com/missionweaveprotocol/missionweaveprotocol/main/assets/brand/missionweaveprotocol-icon.svg" width="160" alt="Icono de MissionWeaveProtocol">
</p>

<p align="center">
  <strong><a href="https://missionweaveprotocol.github.io/">Sitio web y documentación oficiales</a></strong>
</p>

MissionWeaveProtocol Go SDK ofrece bindings de Go basados primero en schemas para
[MissionWeaveProtocol](https://github.com/missionweaveprotocol/missionweaveprotocol) 0.1. El Go
module es `github.com/missionweaveprotocol/go-sdk` y su package raíz es `missionweaveprotocol`.

Esta versión demuestra únicamente **conformidad con esquemas y vectores**. No afirma conformidad de
comportamiento para un Core autoritativo, Agent runtime, Worker Scheduler, Group gateway,
persistencia ni para la máquina de estados completa de Mission/WorkItem.

## Compatibilidad del protocolo

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

Las versiones del SDK y del protocolo son independientes.
[`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) registra el commit exacto del protocolo y los SHA-256
digest de los schemas y conformance vectors incluidos.

## Requisitos e instalación

Se requiere Go 1.24 o posterior.

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## Capacidades incluidas

- protocol pin byte-exact, 21 schemas Draft 2020-12 y 52 conformance vectors embebidos;
- verificación de los digest de schemas, conformance y del bundle combinado;
- análisis JSON UTF-8 estricto con rechazo recursivo de members duplicados;
- resolución offline por `$id`, format assertions y patterns compatibles con ECMAScript;
- `SchemaCatalog` sobre el `fs.FS` embebido o proporcionado por el caller;
- runner de 52 vectors y comando `missionweaveprotocol-conformance`;
- canonicalization JSON RFC 8785 e identificadores `sha256:`;
- firma y verificación Ed25519 con base64url sin padding;
- payloads de firma que excluyen solo el member `signature` superior;
- `FrameCodec` generic, schema-validating y canonical para WebSocket frames.

## Verificar el protocol bundle embebido

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

## Validar un documento de protocolo

```go
catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
if err != nil {
    log.Fatal(err)
}

if err := catalog.Validate("command.schema.json", commandJSON); err != nil {
    log.Fatal(err)
}
```

`NewSchemaCatalog(source fs.FS)` ofrece la misma Interface para un protocol checkout o release
bundle descomprimido. Todos los schemas se registran por `$id` antes de compilarse; las referencias
sin resolver nunca recurren a la red.

## Codificar y decodificar WebSocket frames

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

`DecodeFrame` rechaza UTF-8 malformado, JSON members duplicados, frame variants desconocidas,
campos extra y contenido inválido según el schema. `EncodeFrame` valida antes de devolver JSON
canonical RFC 8785.

## Canonicalizar, calcular hash y firmar

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`, `CanonicalHash` y la Interface de document signing reciben JSON bytes y no
aplican conversiones personalizadas a valores Go como `time.Time`. `MarshalCanonicalJSON` es una
función de conveniencia explícita que usa el marshaling estándar de `encoding/json` antes de JCS.
`SignDocument` y `VerifyDocument` eliminan el member `signature` superior antes de canonicalizar;
los members anidados con el mismo nombre siguen firmados.

## Ejecutar conformance

Contra el protocol bundle embebido:

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

Contra un protocol checkout o release bundle:

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

El éxito muestra `52/52 conformance vectors passed`. El comando termina con un estado distinto de
cero ante un validity mismatch, un vector malformado, un recurso ausente o un error de compilación
del schema.

## Ejemplos y desarrollo

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

El CI gate también verifica formatting, canonical naming, conformance embedded y checkout, y un
compiled binary resource smoke test.

## Alcance

El repository normativo del protocolo sigue siendo la source of truth. Este SDK no copia el server,
los database adapters, el scheduling algorithm, el local runtime ni los internal projection models
de la implementación de referencia de Python. Las futuras funciones de runtime requerirán trabajo
independiente de behavioral conformance y se documentarán por separado.

## Licencia

Licenciado bajo [Apache-2.0](LICENSE).
