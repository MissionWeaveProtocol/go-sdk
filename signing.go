package missionweaveprotocol

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// SignBytes signs bytes with Ed25519 and returns unpadded base64url.
func SignBytes(privateKey ed25519.PrivateKey, message []byte) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", fmt.Errorf("Ed25519 private key must contain %d bytes", ed25519.PrivateKeySize)
	}
	signature := ed25519.Sign(privateKey, message)
	return base64.RawURLEncoding.EncodeToString(signature), nil
}

// VerifyBytes verifies an unpadded base64url Ed25519 signature.
func VerifyBytes(publicKey ed25519.PublicKey, message []byte, signature string) (bool, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return false, fmt.Errorf("Ed25519 public key must contain %d bytes", ed25519.PublicKeySize)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("decode Ed25519 signature: %w", err)
	}
	if len(decoded) != ed25519.SignatureSize {
		return false, nil
	}
	return ed25519.Verify(publicKey, message, decoded), nil
}

// DocumentSigningPayload removes the top-level signature member and canonicalizes the remaining
// JSON object.
func DocumentSigningPayload(document []byte) ([]byte, error) {
	value, err := DecodeJSON(document)
	if err != nil {
		return nil, err
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, errors.New("signed protocol document must be a JSON object")
	}
	delete(object, "signature")
	serialized, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("marshal document signing payload: %w", err)
	}
	return CanonicalizeJSON(serialized)
}

// SignDocument signs a protocol document after excluding its top-level signature member.
func SignDocument(privateKey ed25519.PrivateKey, document []byte) (string, error) {
	payload, err := DocumentSigningPayload(document)
	if err != nil {
		return "", err
	}
	return SignBytes(privateKey, payload)
}

// VerifyDocument verifies a protocol document after excluding its top-level signature member.
func VerifyDocument(publicKey ed25519.PublicKey, document []byte, signature string) (bool, error) {
	payload, err := DocumentSigningPayload(document)
	if err != nil {
		return false, err
	}
	return VerifyBytes(publicKey, payload, signature)
}
