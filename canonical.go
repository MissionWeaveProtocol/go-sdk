package missionweaveprotocol

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
)

// CanonicalizeJSON returns the RFC 8785 JSON Canonicalization Scheme representation of one strict
// JSON document.
func CanonicalizeJSON(document []byte) ([]byte, error) {
	if _, err := DecodeJSON(document); err != nil {
		return nil, err
	}
	canonical, err := jcs.Transform(document)
	if err != nil {
		return nil, fmt.Errorf("canonicalize JSON document: %w", err)
	}
	return canonical, nil
}

// MarshalCanonicalJSON serializes a Go value and returns its RFC 8785 representation.
func MarshalCanonicalJSON(value any) ([]byte, error) {
	document, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal JSON document: %w", err)
	}
	return CanonicalizeJSON(document)
}

// CanonicalHash returns a lowercase SHA-256 content identifier over canonical JSON bytes.
func CanonicalHash(document []byte) (string, error) {
	canonical, err := CanonicalizeJSON(document)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
