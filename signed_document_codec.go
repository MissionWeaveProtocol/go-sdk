package missionweaveprotocol

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// SignedDocumentKind explicitly selects one of the nine signature-required protocol profiles.
type SignedDocumentKind string

const (
	SignedDocumentAgentCard        SignedDocumentKind = "agent-card"
	SignedDocumentApproval         SignedDocumentKind = "approval"
	SignedDocumentArtifact         SignedDocumentKind = "artifact"
	SignedDocumentCommand          SignedDocumentKind = "command"
	SignedDocumentContextPackage   SignedDocumentKind = "context-package"
	SignedDocumentEvent            SignedDocumentKind = "event"
	SignedDocumentEvidence         SignedDocumentKind = "evidence"
	SignedDocumentExtensionProfile SignedDocumentKind = "extension-profile"
	SignedDocumentGroupSnapshot    SignedDocumentKind = "group-snapshot"
)

// SigningKey is the sole application adapter used by SignedDocumentCodec when signing.
type SigningKey interface {
	Algorithm() string
	KeyID() string
	Sign(message []byte) ([]byte, error)
}

type signedDocumentProfile struct {
	protectedTimePointer  string
	schemaName            string
	expectedSignerRule    expectedSignerRule
	expectedSignerPointer string
}

type expectedSignerRule uint8

const (
	expectPrincipalObject expectedSignerRule = iota
	expectAgentID
	expectServicePrincipal
)

var signedDocumentProfiles = map[SignedDocumentKind]signedDocumentProfile{
	SignedDocumentAgentCard: {
		protectedTimePointer: "/issuedAt", schemaName: "agent-card.schema.json",
		expectedSignerRule: expectServicePrincipal,
	},
	SignedDocumentApproval: {
		protectedTimePointer: "/occurredAt", schemaName: "approval.schema.json",
		expectedSignerRule: expectPrincipalObject, expectedSignerPointer: "/approver",
	},
	SignedDocumentArtifact: {
		protectedTimePointer: "/createdAt", schemaName: "artifact.schema.json",
		expectedSignerRule: expectAgentID, expectedSignerPointer: "/producer/agentId",
	},
	SignedDocumentCommand: {
		protectedTimePointer: "/issuedAt", schemaName: "command.schema.json",
		expectedSignerRule: expectPrincipalObject, expectedSignerPointer: "/actor",
	},
	SignedDocumentContextPackage: {
		protectedTimePointer: "/generatedAt", schemaName: "context-package.schema.json",
		expectedSignerRule: expectPrincipalObject, expectedSignerPointer: "/generatedBy",
	},
	SignedDocumentEvent: {
		protectedTimePointer: "/occurredAt", schemaName: "event.schema.json",
		expectedSignerRule: expectPrincipalObject, expectedSignerPointer: "/acceptedBy",
	},
	SignedDocumentEvidence: {
		protectedTimePointer: "/createdAt", schemaName: "evidence.schema.json",
		expectedSignerRule: expectPrincipalObject, expectedSignerPointer: "/generatedBy",
	},
	SignedDocumentExtensionProfile: {
		protectedTimePointer: "/approvedAt", schemaName: "extension-profile.schema.json",
		expectedSignerRule: expectPrincipalObject, expectedSignerPointer: "/approvedBy",
	},
	SignedDocumentGroupSnapshot: {
		protectedTimePointer: "/createdAt", schemaName: "group-snapshot.schema.json",
		expectedSignerRule: expectPrincipalObject, expectedSignerPointer: "/createdBy",
	},
}

// SignedDocumentCodec owns profile selection, schema validation, canonicalization, and signature
// envelope construction for signed protocol documents.
type SignedDocumentCodec struct {
	catalog *SchemaCatalog
}

// NewSignedDocumentCodec builds a codec over the exact schemas embedded in this SDK build.
func NewSignedDocumentCodec() (*SignedDocumentCodec, error) {
	catalog, err := NewEmbeddedSchemaCatalog()
	if err != nil {
		return nil, fmt.Errorf("build signed document codec: %w", err)
	}
	return &SignedDocumentCodec{catalog: catalog}, nil
}

// Sign signs one unsigned JSON object under an explicitly selected signed-document profile.
// The caller's object is never mutated.
func (codec *SignedDocumentCodec) Sign(
	kind SignedDocumentKind,
	unsignedDocument map[string]any,
	signingKey SigningKey,
) (map[string]any, error) {
	profile, ok := signedDocumentProfiles[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported signed document kind %q", kind)
	}
	if unsignedDocument == nil {
		return nil, errors.New("unsigned document must be a JSON object")
	}
	if _, exists := unsignedDocument["signature"]; exists {
		return nil, errors.New("unsigned document already has a top-level signature")
	}
	if signingKey == nil {
		return nil, errors.New("SigningKey must not be nil")
	}
	if signingKey.Algorithm() != "Ed25519" || signingKey.KeyID() == "" {
		return nil, errors.New("SigningKey must identify one Ed25519 key")
	}

	cloned, err := cloneJSONValue(unsignedDocument)
	if err != nil {
		return nil, fmt.Errorf("unsigned document is outside the JSON data model: %w", err)
	}
	unsigned := cloned.(map[string]any)
	protectedTime, ok := jsonPointer(unsigned, profile.protectedTimePointer).(string)
	if !ok {
		return nil, errors.New("protected signed time must be a string")
	}
	if _, err := parseProtocolRFC3339(protectedTime); err != nil {
		return nil, fmt.Errorf("protected signed time is invalid: %w", err)
	}
	if !strings.HasSuffix(protectedTime, "Z") {
		return nil, errors.New("protected signed time must use uppercase Z")
	}
	signingBytes, err := MarshalCanonicalJSON(unsigned)
	if err != nil {
		return nil, fmt.Errorf("canonicalize signed document input: %w", err)
	}
	signatureBytes, err := signingKey.Sign(append([]byte(nil), signingBytes...))
	if err != nil {
		return nil, fmt.Errorf("SigningKey failed: %w", err)
	}
	if len(signatureBytes) != 64 {
		return nil, fmt.Errorf("SigningKey returned %d signature bytes; expected 64", len(signatureBytes))
	}
	if err := strictEd25519Point(signatureBytes[:32], true); err != nil {
		return nil, fmt.Errorf("SigningKey returned invalid signature R: %w", err)
	}
	if littleEndianInteger(signatureBytes[32:]).Cmp(ed25519Order) >= 0 {
		return nil, errors.New("SigningKey returned signature S outside the Ed25519 scalar range")
	}
	unsigned["signature"] = map[string]any{
		"algorithm": "Ed25519",
		"createdAt": protectedTime,
		"keyId":     signingKey.KeyID(),
		"value":     base64.RawURLEncoding.EncodeToString(signatureBytes),
	}
	if err := codec.catalog.validateValue(profile.schemaName, unsigned); err != nil {
		return nil, err
	}
	return unsigned, nil
}

func jsonPointer(document any, pointer string) any {
	if len(pointer) < 2 || pointer[0] != '/' {
		return nil
	}
	current := document
	for _, encoded := range splitJSONPointer(pointer[1:]) {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		value, exists := object[encoded]
		if !exists {
			return nil
		}
		current = value
	}
	return current
}

func splitJSONPointer(pointer string) []string {
	parts := make([]string, 0, 1)
	start := 0
	for index := 0; index <= len(pointer); index++ {
		if index != len(pointer) && pointer[index] != '/' {
			continue
		}
		part := pointer[start:index]
		decoded := make([]byte, 0, len(part))
		for offset := 0; offset < len(part); offset++ {
			if part[offset] == '~' && offset+1 < len(part) {
				switch part[offset+1] {
				case '0':
					decoded = append(decoded, '~')
					offset++
					continue
				case '1':
					decoded = append(decoded, '/')
					offset++
					continue
				}
			}
			decoded = append(decoded, part[offset])
		}
		parts = append(parts, string(decoded))
		start = index + 1
	}
	return parts
}

func cloneJSONValue(value any) (any, error) {
	switch typed := value.(type) {
	case nil, bool:
		return typed, nil
	case string:
		if !utf8.ValidString(typed) {
			return nil, errors.New("string is not valid UTF-8")
		}
		return typed, nil
	case json.Number:
		if _, err := json.Marshal(typed); err != nil {
			return nil, fmt.Errorf("invalid JSON number %q", typed)
		}
		return typed, nil
	case float64:
		if math.IsInf(typed, 0) || math.IsNaN(typed) {
			return nil, errors.New("number is not finite")
		}
		return typed, nil
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			cloned, err := cloneJSONValue(item)
			if err != nil {
				return nil, fmt.Errorf("array item %d: %w", index, err)
			}
			result[index] = cloned
		}
		return result, nil
	case map[string]any:
		result := make(map[string]any, len(typed))
		for name, item := range typed {
			if !utf8.ValidString(name) {
				return nil, errors.New("object member name is not valid UTF-8")
			}
			cloned, err := cloneJSONValue(item)
			if err != nil {
				return nil, fmt.Errorf("object member %q: %w", name, err)
			}
			result[name] = cloned
		}
		return result, nil
	default:
		return nil, fmt.Errorf("value has coercive Go type %T", value)
	}
}
