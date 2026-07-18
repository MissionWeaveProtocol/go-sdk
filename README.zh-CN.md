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
Schema 优先的 Go 绑定。Go 模块为 `github.com/missionweaveprotocol/go-sdk`，根包
为 `missionweaveprotocol`。

本版本仅证明 **Schema 和向量符合性**。它不声明权威 Core、Agent 运行时、
Worker Scheduler、Group 网关、持久化或完整 Mission/工作项状态机的行为符合性。

## 协议兼容性

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

SDK 和协议独立进行版本管理。[`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) 记录精确的协议提交，
以及内置 Schema 和符合性向量的 SHA-256 摘要。

## 要求和安装

需要 Go 1.24 或更高版本。

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## 已包含的能力

- 按原始字节嵌入的协议锁定信息、21 个 Draft 2020-12 Schema 和 52 个符合性向量；
- 验证 Schema、符合性向量和组合协议包的摘要；
- 严格的 UTF-8 JSON 解析，并递归拒绝重复成员；
- 通过 `$id` 离线解析 Schema，并支持格式断言和 ECMAScript 兼容模式；
- 使用内置或调用方提供的 `fs.FS` 的 `SchemaCatalog`；
- 包含 52 个向量的符合性运行器和 `missionweaveprotocol-conformance` 命令；
- RFC 8785 JSON 规范化和 `sha256:` 内容标识符；
- 使用无填充 base64url 值的 Ed25519 签名和验证；
- 签名载荷仅排除顶层 `signature` 成员；
- 用于 WebSocket 帧的通用、通过 Schema 验证且输出规范格式的 `FrameCodec`。

## 验证嵌入的协议包

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

`NewSchemaCatalog(source fs.FS)` 为已解压的协议检出目录或发行包提供相同的
接口。所有 Schema 都会在编译前按 `$id` 注册；未解析的引用绝不会回退到网络。

## 编解码 WebSocket 帧

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

`DecodeFrame` 会拒绝格式错误的 UTF-8、重复 JSON 成员、未知帧变体、额外字段和
不符合 Schema 的内容。`EncodeFrame` 会先验证，再返回规范化的 RFC 8785 JSON。

## 规范化、计算哈希并签名

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`、`CanonicalHash` 和文档签名接口接受 JSON 字节，且不会
对 `time.Time` 等 Go 值执行自定义转换。`MarshalCanonicalJSON` 是显式的便利函数，会先使用
标准 `encoding/json` 序列化，再执行 JCS。`SignDocument` 和 `VerifyDocument` 会在
规范化前移除顶层 `signature` 成员；同名的嵌套成员仍会被签名。

## 运行符合性测试

针对嵌入的协议包运行：

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

针对协议检出目录或发行包运行：

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

成功时会报告 `52/52 conformance vectors passed`。如果有效性不匹配、向量格式错误、
资源缺失或 Schema 编译失败，命令将以非零状态退出。

## 示例和开发

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

CI 门禁还会验证格式、规范命名、内置和检出目录两种符合性模式，以及
编译后二进制文件的资源冒烟测试。

## 范围

规范性协议仓库始终是权威来源。本 SDK 有意不复制 Python 参考实现的服务器、数据库
适配器、调度算法、本地运行时或内部投影模型。未来的运行时
功能需要独立的行为符合性工作，并会单独记录。

## 许可证

采用 [Apache-2.0](LICENSE) 许可证。
