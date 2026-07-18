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
Schema 優先的 Go 綁定。Go 模組為 `github.com/missionweaveprotocol/go-sdk`，根套件
為 `missionweaveprotocol`。

本版本僅證明 **Schema 和向量符合性**。它不宣稱權威 Core、Agent
runtime、Worker Scheduler、Group gateway、持久化或完整 Mission/工作項狀態機的行為符合性。

## 協定相容性

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

SDK 和協定採用獨立版本。[`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) 記錄精確的協定提交，
以及內建 Schema 和符合性向量的 SHA-256 摘要。

## 需求和安裝

需要 Go 1.24 或更新版本。

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## 已包含的能力

- 依原始位元組嵌入的協定鎖定資訊、21 個 Draft 2020-12 Schema 和 52 個符合性向量；
- 驗證 Schema、符合性向量和組合協定包的摘要；
- 嚴格的 UTF-8 JSON 解析，並遞迴拒絕重複成員；
- 透過 `$id` 離線解析 Schema，並支援格式斷言和 ECMAScript 相容模式；
- 使用內建或呼叫端提供的 `fs.FS` 的 `SchemaCatalog`；
- 包含 52 個向量的符合性執行器和 `missionweaveprotocol-conformance` 指令；
- RFC 8785 JSON 規範化和 `sha256:` 內容識別子；
- 使用無填充 base64url 值的 Ed25519 簽署和驗證；
- 簽署載荷僅排除頂層 `signature` 成員；
- `SignedDocumentCodec` 涵蓋全部 22 個密碼學案例和 58 項評估；
- 用於 WebSocket frame 的通用、通過 Schema 驗證且輸出規範格式的 `FrameCodec`。

## 驗證嵌入的協定包

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

`NewSchemaCatalog(source fs.FS)` 為已解壓縮的協定檢出目錄或發行包提供相同的
介面。所有 Schema 都會在編譯前依 `$id` 註冊；未解析的參照絕不會回退到網路。

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

`DecodeFrame` 會拒絕格式錯誤的 UTF-8、重複 JSON 成員、未知 frame 變體、額外欄位和
不符合 Schema 的內容。`EncodeFrame` 會先驗證，再傳回規範化的 RFC 8785 JSON。

## 規範化、計算雜湊並簽署

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`、`CanonicalHash` 和文件簽署介面接受 JSON 位元組，且不會
對 `time.Time` 等 Go 值執行自訂轉換。`MarshalCanonicalJSON` 是明確的便利函式，會先使用
標準 `encoding/json` 序列化，再執行 JCS。`SignDocument` 和 `VerifyDocument` 會在
規範化前移除頂層 `signature` 成員；同名的巢狀成員仍會被簽署。

## 簽名並驗證 Signed Document

`SignedDocumentCodec` 依序實作密碼學驗證流程，而且僅接受九種明確的文件 kind：

```go
codec, err := missionweaveprotocol.NewSignedDocumentCodec()
signed, err := codec.Sign(missionweaveprotocol.SignedDocumentCommand, unsigned, signingKey)
verified, err := codec.Verify(missionweaveprotocol.SignedDocumentCommand, raw, keyResolver)
fmt.Println(signed["signature"], verified.SigningHash(), verified.ResolvedKey().Principal())
```

`SigningKey` 是唯一的簽名 adapter。`KeyResolver` 接收 `KeyResolutionRequest`，而且必須回傳
明確宣告 `KeyRegistryOrganizationWide` 完整性的 `KeyRegistrySnapshot`；局部或未宣告完整性的
Registry snapshot 會 fail closed。驗證錯誤對 peer 僅公開穩定的 `WireCode()`，而
`ProtectedDiagnostic()` 會為本機維運保留第一個失敗階段與原因。可執行的測試 fixture 範例見
[`examples/sign`](examples/sign)。

## 執行符合性測試

針對嵌入的協定包執行：

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

針對協定檢出目錄或發行包執行：

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

成功時會回報 `52/52 conformance vectors passed`。若有效性不相符、向量格式錯誤、
資源缺失或 Schema 編譯失敗，指令將以非零狀態結束。

## 範例和開發

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

CI 門檻還會驗證格式、規範命名、內建和檢出目錄兩種符合性模式，以及
編譯後二進位檔的資源煙霧測試。

## 範圍

規範性協定儲存庫始終是權威來源。本 SDK 刻意不複製 Python 參考實作的伺服器、資料庫
轉接器、排程演算法、本機 runtime 或內部投影模型。未來的 runtime
功能需要獨立的行為符合性工作，並會另行記錄。

## 授權條款

採用 [Apache-2.0](LICENSE) 授權條款。
