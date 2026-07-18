[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) |
[日本語](README.ja.md) | [Español](README.es.md) | **Français** |
[Deutsch](README.de.md)

# SDK Go MissionWeaveProtocol

<p align="center">
  <img src="https://raw.githubusercontent.com/missionweaveprotocol/missionweaveprotocol/main/assets/brand/missionweaveprotocol-icon.svg" width="160" alt="Icône MissionWeaveProtocol">
</p>

<p align="center">
  <strong><a href="https://missionweaveprotocol.github.io/">Site officiel et documentation</a></strong>
</p>

Le MissionWeaveProtocol Go SDK fournit des bindings Go schema-first pour
[MissionWeaveProtocol](https://github.com/missionweaveprotocol/missionweaveprotocol) 0.1. Le Go
module est `github.com/missionweaveprotocol/go-sdk` et son package racine est
`missionweaveprotocol`.

Cette version démontre uniquement une **conformité limitée aux schémas et aux vecteurs**. Elle ne
revendique aucune conformité comportementale pour un Core faisant autorité, un Agent runtime, un
Worker Scheduler, un Group gateway, la persistance ou la machine à états complète Mission/WorkItem.

## Compatibilité du protocole

| Go SDK  | MissionWeaveProtocol |
| ------- | -------------------- |
| `0.1.x` | `0.1`                |

Les versions du SDK et du protocole sont indépendantes.
[`PROTOCOL_PIN.json`](PROTOCOL_PIN.json) enregistre le commit exact du protocole ainsi que les
empreintes SHA-256 des schémas et des vecteurs de conformité embarqués.

## Prérequis et installation

Go 1.24 ou une version ultérieure est requis.

```bash
go get github.com/missionweaveprotocol/go-sdk@latest
```

## Capacités incluses

- protocol pin byte-exact, 21 schemas Draft 2020-12 et 52 conformance vectors embarqués ;
- vérification des digest de schemas, de conformance et du bundle combiné ;
- parsing JSON UTF-8 strict avec rejet récursif des members dupliqués ;
- résolution offline par `$id`, format assertions et patterns compatibles ECMAScript ;
- `SchemaCatalog` basé sur le `fs.FS` embarqué ou fourni par l'appelant ;
- runner de 52 vectors et commande `missionweaveprotocol-conformance` ;
- canonicalization JSON RFC 8785 et identifiants `sha256:` ;
- signature et vérification Ed25519 en base64url sans padding ;
- payload de signature excluant uniquement le member `signature` de premier niveau ;
- `FrameCodec` generic, schema-validating et canonical pour les WebSocket frames.

## Vérifier le protocol bundle embarqué

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

## Valider un document de protocole

```go
catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
if err != nil {
    log.Fatal(err)
}

if err := catalog.Validate("command.schema.json", commandJSON); err != nil {
    log.Fatal(err)
}
```

`NewSchemaCatalog(source fs.FS)` fournit la même Interface pour un protocol checkout ou release
bundle décompressé. Tous les schemas sont enregistrés par `$id` avant compilation ; les références
non résolues ne basculent jamais vers le réseau.

## Encoder et décoder les WebSocket frames

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

`DecodeFrame` rejette l'UTF-8 malformé, les JSON members dupliqués, les frame variants inconnues,
les champs supplémentaires et le contenu non conforme au schema. `EncodeFrame` valide avant de
renvoyer du JSON canonical RFC 8785.

## Canonicaliser, calculer le hash et signer

```go
canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
hash, err := missionweaveprotocol.CanonicalHash(document)
signature, err := missionweaveprotocol.SignDocument(privateKey, document)
verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
```

`CanonicalizeJSON`, `CanonicalHash` et l'Interface de document signing acceptent des JSON bytes et
n'appliquent aucune conversion personnalisée aux valeurs Go comme `time.Time`.
`MarshalCanonicalJSON` est une fonction de commodité explicite qui applique le marshaling standard
de `encoding/json` avant JCS. `SignDocument` et `VerifyDocument` retirent le member `signature` de
premier niveau avant canonicalization ; les members imbriqués de même nom restent signés.

## Exécuter la conformance

Avec le protocol bundle embarqué :

```bash
go run github.com/missionweaveprotocol/go-sdk/cmd/missionweaveprotocol-conformance@latest
```

Avec un protocol checkout ou release bundle :

```bash
go run ./cmd/missionweaveprotocol-conformance --root ../missionweaveprotocol
```

En cas de succès, la commande affiche `52/52 conformance vectors passed`. Elle retourne un état non
nul en cas de validity mismatch, de vector malformé, de ressource manquante ou d'erreur de
compilation de schema.

## Exemples et développement

```bash
go run ./examples/validate
go run ./examples/sign
go run ./internal/cmd/repository-policy
go test -race ./...
go vet ./...
go build ./...
```

Le CI gate vérifie aussi le formatting, le canonical naming, la conformance embedded et checkout,
ainsi qu'un compiled binary resource smoke test.

## Périmètre

Le repository normatif du protocole reste la source of truth. Ce SDK ne copie volontairement ni le
server, ni les database adapters, ni le scheduling algorithm, ni le local runtime, ni les internal
projection models de l'implémentation Python de référence. Les futures fonctions de runtime
nécessiteront un travail distinct de behavioral conformance et seront documentées séparément.

## Licence

Sous licence [Apache-2.0](LICENSE).
