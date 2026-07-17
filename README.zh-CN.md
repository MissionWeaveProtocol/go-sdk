[English](README.md) | **简体中文** | [繁體中文](README.zh-TW.md) |
[日本語](README.ja.md) | [Español](README.es.md) | [Français](README.fr.md) |
[Deutsch](README.de.md)

# MissionWeaveProtocol Go SDK

<p align="center">
  <img src="https://raw.githubusercontent.com/missionweaveprotocol/missionweaveprotocol/main/assets/brand/missionweaveprotocol-icon.svg" width="160" alt="MissionWeaveProtocol 图标">
</p>

<p align="center">
  <strong><a href="https://missionweaveprotocol.github.io/">官方网站和文档</a></strong>
</p>

MissionWeaveProtocol Go SDK 为
[MissionWeaveProtocol](https://github.com/missionweaveprotocol/missionweaveprotocol) 0.1 提供
schema-first Go bindings。Go module 为 `github.com/missionweaveprotocol/go-sdk`，根 package
为 `missionweaveprotocol`。

本版本仅证明 **schema-and-vector conformance**。它不声明 authoritative Core、Agent runtime、
Worker Scheduler、Group gateway、持久化或完整 Mission/WorkItem 状态机的 behavioral
conformance。

## 协议兼容性

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

SDK 和协议独立版本化。[`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) 记录精确的协议 commit，
以及 vendored schema 和 conformance vector 的 SHA-256 digest。

## 要求和安装

需要 Go 1.24 或更高版本。

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## 已包含的能力

- 按原始字节嵌入的 protocol pin、21 个 Draft 2020-12 schema 和 43 个 conformance vector；
- 验证 schema、conformance 和组合 bundle digest；
- 严格的 UTF-8 JSON 解析，并递归拒绝重复 member；
- 离线 `$id` schema 解析、format assertion 和 ECMAScript 兼容 pattern；
- 使用 embedded 或调用方提供的 `fs.FS` 的 `SchemaCatalog`；
- 43-vector conformance runner 和 `missionweaveprotocol-conformance` 命令；
- RFC 8785 JSON canonicalization 和 `sha256:` content identifier；
- 使用无 padding base64url 值的 Ed25519 签名和验证；
- 签名 payload 仅排除顶层 `signature` member；
- 用于 WebSocket frame 的 generic、schema-validating、canonical `FrameCodec`。

## 验证嵌入的协议 bundle

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

## 验证协议文档

```go
catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
if err != nil {
    log.Fatal(err)
}

if err := catalog.Validate("command.schema.json", commandJSON); err != nil {
    log.Fatal(err)
}
```

`NewSchemaCatalog(source fs.FS)` 为已解压的协议 checkout 或 release bundle 提供相同的
Interface。所有 schema 都会在编译前按 `$id` 注册；未解析的引用绝不会回退到网络。

## 编解码 WebSocket frame

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

`DecodeFrame` 会拒绝格式错误的 UTF-8、重复 JSON member、未知 frame variant、额外字段和
不符合 schema 的内容。`EncodeFrame` 会先验证，再返回 canonical RFC 8785 JSON。

## Canonicalize、hash 和签名

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`、`CanonicalHash` 和 document-signing Interface 接受 JSON bytes，且不会
对 `time.Time` 等 Go 值执行自定义转换。`MarshalCanonicalJSON` 是显式的便利函数，会先使用
标准 `encoding/json` marshaling，再执行 JCS。`SignDocument` 和 `VerifyDocument` 会在
canonicalization 前移除顶层 `signature` member；同名的嵌套 member 仍会被签名。

## 运行 conformance

针对嵌入的协议 bundle 运行：

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

针对协议 checkout 或 release bundle 运行：

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

成功时会报告 `43/43 conformance vectors passed`。如果 validity 不匹配、vector 格式错误、
资源缺失或 schema 编译失败，命令将以非零状态退出。

## 示例和开发

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

CI gate 还会验证格式、canonical naming、embedded 和 checkout 两种 conformance，以及
compiled binary resource smoke test。

## 范围

规范性协议仓库始终是权威来源。本 SDK 有意不复制 Python 参考实现的 server、database
adapter、scheduling algorithm、local runtime 或 internal projection model。未来的 runtime
功能需要独立的 behavioral conformance 工作，并会单独记录。

## 许可证

采用 [Apache-2.0](LICENSE) 许可证。
