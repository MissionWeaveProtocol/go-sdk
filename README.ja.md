[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) |
**日本語** | [Español](README.es.md) | [Français](README.fr.md) |
[Deutsch](README.de.md)

# MissionWeaveProtocol Go SDK

<p align="center">
  <img src="https://raw.githubusercontent.com/missionweaveprotocol/missionweaveprotocol/main/assets/brand/missionweaveprotocol-icon.svg" width="160" alt="MissionWeaveProtocol アイコン">
</p>

<p align="center">
  <strong><a href="https://missionweaveprotocol.github.io/">公式サイトとドキュメント</a></strong>
</p>

MissionWeaveProtocol Go SDK は、
[MissionWeaveProtocol](https://github.com/missionweaveprotocol/missionweaveprotocol) 0.1 向けの
schema-first Go bindings を提供します。Go module は `github.com/missionweaveprotocol/go-sdk`、
ルート package は `missionweaveprotocol` です。

このリリースが示すのは **schema-and-vector conformance** のみです。authoritative Core、
Agent runtime、Worker Scheduler、Group gateway、永続化、または完全な Mission/WorkItem
状態機械に対する behavioral conformance は表明しません。

## プロトコル互換性

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

SDK とプロトコルは独立してバージョン管理されます。
[`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) は、正確なプロトコル commit と、vendored schema
および conformance vector の SHA-256 digest を記録します。

## 要件とインストール

Go 1.24 以降が必要です。

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## 含まれる機能

- バイト単位で同一の embedded protocol pin、21 個の Draft 2020-12 schema、43 個の conformance vector；
- schema、conformance、結合 bundle digest の検証；
- 重複 member を再帰的に拒否する厳格な UTF-8 JSON parser；
- format assertion と ECMAScript 互換 pattern を備えたオフライン `$id` schema 解決；
- embedded または呼び出し側が提供する `fs.FS` を使う `SchemaCatalog`；
- 43-vector conformance runner と `missionweaveprotocol-conformance` コマンド；
- RFC 8785 JSON canonicalization と `sha256:` content identifier；
- padding なし base64url を使う Ed25519 署名と検証；
- トップレベルの `signature` member だけを除外する署名 payload；
- WebSocket frame 向けの generic、schema-validating、canonical `FrameCodec`。

## Embedded protocol bundle の検証

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

## プロトコル文書の検証

```go
catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
if err != nil {
    log.Fatal(err)
}

if err := catalog.Validate("command.schema.json", commandJSON); err != nil {
    log.Fatal(err)
}
```

`NewSchemaCatalog(source fs.FS)` は、展開済みのプロトコル checkout または release bundle
に同じ Interface を提供します。すべての schema はコンパイル前に `$id` で登録され、
未解決の参照がネットワークへフォールバックすることはありません。

## WebSocket frame の encode と decode

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

`DecodeFrame` は、不正な UTF-8、重複 JSON member、未知の frame variant、余分なフィールド、
schema-invalid content を拒否します。`EncodeFrame` は検証後に canonical RFC 8785 JSON
を返します。

## Canonicalize、hash、署名

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`、`CanonicalHash`、document-signing Interface は JSON bytes を受け取り、
`time.Time` などの Go 値へ独自変換を適用しません。`MarshalCanonicalJSON` は標準の
`encoding/json` marshaling の後に JCS を実行する明示的な convenience function です。
`SignDocument` と `VerifyDocument` は canonicalization の前にトップレベルの `signature`
member を削除します。同名のネストされた member は署名対象のままです。

## Conformance の実行

Embedded protocol bundle に対して実行します：

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

プロトコル checkout または release bundle に対して実行します：

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

成功時は `43/43 conformance vectors passed` と表示されます。validity mismatch、malformed
vector、resource 不足、schema compile error の場合は非ゼロで終了します。

## サンプルと開発

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

CI gate は formatting、canonical naming、embedded/checkout の両 conformance、compiled
binary resource smoke test も検証します。

## スコープ

規範的プロトコル repository が常に source of truth です。この SDK は Python reference
implementation の server、database adapter、scheduling algorithm、local runtime、internal
projection model を意図的にコピーしません。将来の runtime 機能には独立した behavioral
conformance が必要であり、別途文書化します。

## ライセンス

[Apache-2.0](LICENSE) の下で提供されます。
