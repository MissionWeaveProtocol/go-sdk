// This command demonstrates SignedDocumentCodec with protocol-owned test-only fixtures.
// Production applications must supply their own organization-controlled SigningKey and
// KeyResolver adapters.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"log"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

type fixtureSigningKey struct {
	keyID      string
	privateKey ed25519.PrivateKey
}

func (key fixtureSigningKey) Algorithm() string { return "Ed25519" }

func (key fixtureSigningKey) KeyID() string { return key.keyID }

func (key fixtureSigningKey) Sign(message []byte) ([]byte, error) {
	return ed25519.Sign(key.privateKey, message), nil
}

type fixtureKeyResolver struct {
	registry []byte
}

func (resolver fixtureKeyResolver) Resolve(
	_ missionweaveprotocol.KeyResolutionRequest,
) (missionweaveprotocol.KeyRegistrySnapshot, error) {
	return missionweaveprotocol.KeyRegistrySnapshot{
		Completeness:  missionweaveprotocol.KeyRegistryOrganizationWide,
		RegistryBytes: append([]byte(nil), resolver.registry...),
	}, nil
}

func main() {
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		log.Fatal(err)
	}
	command := readObject("cryptography/vectors/signed-documents/valid/command.json")
	delete(command, "signature")
	keyFixture := readObject("cryptography/keys/signing-coordinator.json")
	seed, err := base64.RawURLEncoding.Strict().DecodeString(keyFixture["seed"].(string))
	if err != nil {
		log.Fatal(err)
	}
	signed, err := codec.Sign(
		missionweaveprotocol.SignedDocumentCommand,
		command,
		fixtureSigningKey{
			keyID:      keyFixture["keyId"].(string),
			privateKey: ed25519.NewKeyFromSeed(seed),
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	signedBytes, err := missionweaveprotocol.MarshalCanonicalJSON(signed)
	if err != nil {
		log.Fatal(err)
	}
	registry, err := missionweaveprotocol.ReadProtocolFile("cryptography/keys/registry-valid.json")
	if err != nil {
		log.Fatal(err)
	}
	verified, err := codec.Verify(
		missionweaveprotocol.SignedDocumentCommand,
		signedBytes,
		fixtureKeyResolver{registry: registry},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(
		"signingHash=%s completeHash=%s principal=%s\n",
		verified.SigningHash(),
		verified.CompleteHash(),
		verified.ResolvedKey().Principal().ID,
	)
}

func readObject(name string) map[string]any {
	raw, err := missionweaveprotocol.ReadProtocolFile(name)
	if err != nil {
		log.Fatal(err)
	}
	value, err := missionweaveprotocol.DecodeJSON(raw)
	if err != nil {
		log.Fatal(err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		log.Fatalf("%s is not a JSON object", name)
	}
	return object
}
