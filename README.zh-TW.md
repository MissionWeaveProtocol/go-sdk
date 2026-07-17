[English](README.md) | [简体中文](README.zh-CN.md) | **繁體中文** |
[日本語](README.ja.md) | [Español](README.es.md) | [Français](README.fr.md) |
[Deutsch](README.de.md)

# MissionWeaveProtocol Go SDK

<p align="center">
  <img src="https://raw.githubusercontent.com/missionweaveprotocol/missionweaveprotocol/main/assets/brand/missionweaveprotocol-icon.svg" width="160" alt="MissionWeaveProtocol 圖示">
</p>

<p align="center">
  <strong><a href="https://missionweaveprotocol.github.io/">官方網站和文件</a></strong>
</p>

MissionWeaveProtocol Go SDK 為
[MissionWeaveProtocol](https://github.com/missionweaveprotocol/missionweaveprotocol) 0.1 提供
schema-first Go bindings。Go module 為 `github.com/missionweaveprotocol/go-sdk`，根 package
為 `missionweaveprotocol`。

本版本僅證明 **schema-and-vector conformance**。它不宣稱 authoritative Core、Agent
runtime、Worker Scheduler、Group gateway、持久化或完整 Mission/WorkItem 狀態機的
behavioral conformance。

## 協定相容性

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

SDK 和協定採用獨立版本。[`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) 記錄精確的協定 commit，
以及 vendored schema 和 conformance vector 的 SHA-256 digest。

## 需求和安裝

需要 Go 1.24 或更新版本。

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## 已包含的能力

- 依原始位元組嵌入的 protocol pin、21 個 Draft 2020-12 schema 和 43 個 conformance vector；
- 驗證 schema、conformance 和組合 bundle digest；
- 嚴格的 UTF-8 JSON 解析，並遞迴拒絕重複 member；
- 離線 `$id` schema 解析、format assertion 和 ECMAScript 相容 pattern；
- 使用 embedded 或呼叫端提供的 `fs.FS` 的 `SchemaCatalog`；
- 43-vector conformance runner 和 `missionweaveprotocol-conformance` 指令；
- RFC 8785 JSON canonicalization 和 `sha256:` content identifier；
- 使用無 padding base64url 值的 Ed25519 簽署和驗證；
- 簽署 payload 僅排除頂層 `signature` member；
- 用於 WebSocket frame 的 generic、schema-validating、canonical `FrameCodec`。

## 驗證嵌入的協定 bundle

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

## 驗證協定文件

```go
catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
if err != nil {
    log.Fatal(err)
}

if err := catalog.Validate("command.schema.json", commandJSON); err != nil {
    log.Fatal(err)
}
```

`NewSchemaCatalog(source fs.FS)` 為已解壓縮的協定 checkout 或 release bundle 提供相同的
Interface。所有 schema 都會在編譯前依 `$id` 註冊；未解析的參照絕不會回退到網路。

## 編解碼 WebSocket frame

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

`DecodeFrame` 會拒絕格式錯誤的 UTF-8、重複 JSON member、未知 frame variant、額外欄位和
不符合 schema 的內容。`EncodeFrame` 會先驗證，再傳回 canonical RFC 8785 JSON。

## Canonicalize、hash 和簽署

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`、`CanonicalHash` 和 document-signing Interface 接受 JSON bytes，且不會
對 `time.Time` 等 Go 值執行自訂轉換。`MarshalCanonicalJSON` 是明確的便利函式，會先使用
標準 `encoding/json` marshaling，再執行 JCS。`SignDocument` 和 `VerifyDocument` 會在
canonicalization 前移除頂層 `signature` member；同名的巢狀 member 仍會被簽署。

## 執行 conformance

針對嵌入的協定 bundle 執行：

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

針對協定 checkout 或 release bundle 執行：

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

成功時會回報 `43/43 conformance vectors passed`。若 validity 不相符、vector 格式錯誤、
資源缺失或 schema 編譯失敗，指令將以非零狀態結束。

## 範例和開發

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

CI gate 還會驗證格式、canonical naming、embedded 和 checkout 兩種 conformance，以及
compiled binary resource smoke test。

## 範圍

規範性協定儲存庫始終是權威來源。本 SDK 刻意不複製 Python 參考實作的 server、database
adapter、scheduling algorithm、local runtime 或 internal projection model。未來的 runtime
功能需要獨立的 behavioral conformance 工作，並會另行記錄。

## 授權條款

採用 [Apache-2.0](LICENSE) 授權條款。
