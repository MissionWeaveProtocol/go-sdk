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
Schema 優先の Go バインディングを提供します。Go モジュールは `github.com/missionweaveprotocol/go-sdk`、
ルートパッケージは `missionweaveprotocol` です。

このリリースが示すのは **Schema とベクトルへの適合**のみです。権威ある Core、
Agent ランタイム、Worker Scheduler、Group ゲートウェイ、永続化、または完全な Mission/WorkItem
状態マシンに対する振る舞いの適合性は表明しません。

## プロトコル互換性

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

SDK とプロトコルは独立してバージョン管理されます。
[`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) は、正確なプロトコルコミットと、同梱 Schema
および適合性ベクトルの SHA-256 ダイジェストを記録します。

## 要件とインストール

Go 1.24 以降が必要です。

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## 含まれる機能

- バイト単位で同一の組み込みプロトコルピン、21 個の Draft 2020-12 Schema、52 個の適合性ベクトル。
- Schema、適合性ベクトル、および結合バンドルのダイジェスト検証。
- 重複メンバーを再帰的に拒否する厳格な UTF-8 JSON パーサー。
- 形式アサーションと ECMAScript 互換パターンを備えたオフライン `$id` Schema 解決。
- 組み込みまたは呼び出し側が提供する `fs.FS` を使う `SchemaCatalog`。
- 52 ベクトルの適合性ランナーと `missionweaveprotocol-conformance` コマンド。
- RFC 8785 JSON 正規化と `sha256:` コンテンツ識別子。
- パディングなし base64url を使う Ed25519 署名と検証。
- トップレベルの `signature` メンバーだけを除外する署名ペイロード。
- 全 22 件の暗号ケースと 58 評価を網羅する `SignedDocumentCodec`。
- WebSocket フレーム向けの汎用的で、Schema 検証と正規化を行う `FrameCodec`。

## 組み込みプロトコルバンドルの検証

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

`NewSchemaCatalog(source fs.FS)` は、展開済みのプロトコルチェックアウトまたはリリースバンドル
に同じインターフェースを提供します。すべての Schema はコンパイル前に `$id` で登録され、
未解決の参照がネットワークへフォールバックすることはありません。

## WebSocket フレームのエンコードとデコード

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

`DecodeFrame` は、不正な UTF-8、重複 JSON メンバー、未知のフレームバリアント、余分なフィールド、
Schema に適合しないコンテンツを拒否します。`EncodeFrame` は検証後に RFC 8785 準拠の正規化 JSON
を返します。

## 正規化、ハッシュ計算、署名

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`、`CanonicalHash`、文書署名インターフェースは JSON バイトを受け取り、
`time.Time` などの Go 値へ独自変換を適用しません。`MarshalCanonicalJSON` は標準の
`encoding/json` によるマーシャリングの後に JCS を実行する明示的な便利関数です。
`SignDocument` と `VerifyDocument` は正規化の前にトップレベルの `signature`
メンバーを削除します。同名のネストされたメンバーは署名対象のままです。

## Signed Document の署名と検証

`SignedDocumentCodec` は暗号検証プロファイルを順序どおりに実装し、明示された 9 種類の
文書 kind だけを受け取ります。

```go
codec, err := missionweaveprotocol.NewSignedDocumentCodec()
signed, err := codec.Sign(missionweaveprotocol.SignedDocumentCommand, unsigned, signingKey)
verified, err := codec.Verify(missionweaveprotocol.SignedDocumentCommand, raw, keyResolver)
fmt.Println(signed["signature"], verified.SigningHash(), verified.ResolvedKey().Principal())
```

署名側の唯一のアダプターは `SigningKey` です。`KeyResolver` は `KeyResolutionRequest` を
受け取り、完全性を `KeyRegistryOrganizationWide` と明示した `KeyRegistrySnapshot` を返す
必要があります。部分的、または完全性が未指定の Agent Registry スナップショットは fail closed
になります。検証エラーは peer には安定した `WireCode()` だけを公開し、
`ProtectedDiagnostic()` はローカル運用向けに最初の失敗段階と理由を保持します。実行可能な
テスト fixture の例は [`examples/sign`](examples/sign) を参照してください。

## 適合性テストの実行

組み込みプロトコルバンドルに対して実行します：

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

プロトコルチェックアウトまたはリリースバンドルに対して実行します：

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

成功時は `52/52 conformance vectors passed` と表示されます。妥当性の不一致、不正な
ベクトル、リソース不足、Schema のコンパイルエラーの場合は非ゼロで終了します。

## サンプルと開発

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

CI ゲートは、フォーマット、正規命名、組み込みバンドルとチェックアウトの両方の適合性、コンパイル済み
バイナリのリソーススモークテストも検証します。

## スコープ

規範的なプロトコルリポジトリが常に信頼できる唯一の情報源です。この SDK は Python リファレンス
実装のサーバー、データベースアダプター、スケジューリングアルゴリズム、ローカルランタイム、内部
プロジェクションモデルを意図的にコピーしません。将来のランタイム機能には独立した振る舞いの適合性検証が必要であり、別途文書化します。

## ライセンス

[Apache-2.0](LICENSE) の下で提供されます。
