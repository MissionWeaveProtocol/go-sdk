package missionweaveprotocol_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"testing"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func TestEd25519MatchesRFC8032VectorOne(t *testing.T) {
	seed, _ := hex.DecodeString("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60")
	wantPublic, _ := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	wantSignature, _ := hex.DecodeString(
		"e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e06522490155" +
			"5fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b",
	)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	if string(publicKey) != string(wantPublic) {
		t.Fatal("RFC 8032 public key mismatch")
	}

	signature, err := missionweaveprotocol.SignBytes(privateKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	if signature != base64.RawURLEncoding.EncodeToString(wantSignature) {
		t.Fatalf("RFC 8032 signature mismatch: %s", signature)
	}
	verified, err := missionweaveprotocol.VerifyBytes(publicKey, nil, signature)
	if err != nil || !verified {
		t.Fatalf("RFC 8032 signature did not verify: verified=%v err=%v", verified, err)
	}
}

func TestDocumentSignatureExcludesOnlyTopLevelSignature(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 1)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	first := []byte(`{"payload":{"signature":"nested"},"signature":{"value":"old"},"value":1}`)
	second := []byte(`{"signature":{"value":"different"},"value":1,"payload":{"signature":"nested"}}`)

	left, err := missionweaveprotocol.SignDocument(privateKey, first)
	if err != nil {
		t.Fatal(err)
	}
	right, err := missionweaveprotocol.SignDocument(privateKey, second)
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatal("top-level signature changed the document signing payload")
	}
	verified, err := missionweaveprotocol.VerifyDocument(publicKey, second, left)
	if err != nil || !verified {
		t.Fatalf("document signature did not verify: verified=%v err=%v", verified, err)
	}
	tampered := []byte(`{"value":2,"payload":{"signature":"nested"}}`)
	verified, err = missionweaveprotocol.VerifyDocument(publicKey, tampered, left)
	if err != nil {
		t.Fatal(err)
	}
	if verified {
		t.Fatal("tampered document verified")
	}
}
