package missionweaveprotocol

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
)

// KeyRegistryCompleteness states the scope over which Registry uniqueness can be proven.
type KeyRegistryCompleteness string

const (
	// KeyRegistryOrganizationWide asserts that the snapshot contains every signing-key binding in
	// the organization, which is required to detect key reuse and aliases.
	KeyRegistryOrganizationWide KeyRegistryCompleteness = "organization-wide"
)

// KeyRegistrySnapshot is raw Registry evidence plus an explicit completeness assertion. The
// codec fails closed unless Completeness is KeyRegistryOrganizationWide.
type KeyRegistrySnapshot struct {
	Completeness  KeyRegistryCompleteness
	RegistryBytes []byte
}

// ExpectedSignerRule describes the signer identity constraint already derived from the document.
type ExpectedSignerRule string

const (
	ExpectedSignerExactPrincipal   ExpectedSignerRule = "exact-principal"
	ExpectedSignerServicePrincipal ExpectedSignerRule = "service-principal"
)

// ExpectedSigner is the signer identity evidence supplied to a KeyResolver request. Principal is
// populated only for ExpectedSignerExactPrincipal.
type ExpectedSigner struct {
	Rule      ExpectedSignerRule
	Principal Principal
}

// ProtectedSignedTime retains both exact timestamp text and its arbitrary-precision instant.
type ProtectedSignedTime struct {
	Text    string
	Instant RFC3339Instant
}

// KeyResolutionRequest gives a resolver enough context to select the correct organization-wide
// Registry without delegating any normative validation to it.
type KeyResolutionRequest struct {
	Kind           SignedDocumentKind
	KeyID          string
	ExpectedSigner ExpectedSigner
	ProtectedTime  ProtectedSignedTime
}

// KeyResolver is the sole application adapter used during Signed Document verification. It must
// return a complete organization Registry snapshot/history; the codec, not the adapter, validates
// and resolves that Registry.
type KeyResolver interface {
	Resolve(request KeyResolutionRequest) (KeyRegistrySnapshot, error)
}

// VerificationStage identifies the first normative Signed Document verification stage reached.
type VerificationStage string

const (
	VerificationParse             VerificationStage = "parse"
	VerificationSchema            VerificationStage = "schema"
	VerificationSignatureEnvelope VerificationStage = "signature-envelope"
	VerificationKeyResolution     VerificationStage = "key-resolution"
	VerificationCanonicalization  VerificationStage = "canonicalization"
	VerificationSignature         VerificationStage = "signature"
	VerificationComplete          VerificationStage = "complete"
)

// WireErrorCode is the non-oracular protocol error exposed to a remote peer.
type WireErrorCode string

const (
	WireProtocolViolation      WireErrorCode = "PROTOCOL_VIOLATION"
	WireSchemaValidationFailed WireErrorCode = "SCHEMA_VALIDATION_FAILED"
	WireAuthInvalidSignature   WireErrorCode = "AUTH_INVALID_SIGNATURE"
)

// VerificationDiagnostic is protected local evidence about the first failing stage. It is kept
// separate from Error(), whose text intentionally exposes only the wire classification.
type VerificationDiagnostic struct {
	stage  VerificationStage
	reason string
}

// Stage returns the first failing semantic verification stage.
func (diagnostic VerificationDiagnostic) Stage() VerificationStage { return diagnostic.stage }

// Reason returns the protected local failure reason.
func (diagnostic VerificationDiagnostic) Reason() string { return diagnostic.reason }

// SignedDocumentVerificationError is a deliberately non-oracular verification failure.
type SignedDocumentVerificationError struct {
	wireCode   WireErrorCode
	diagnostic VerificationDiagnostic
}

// Error exposes only the stable wire classification, never the failing stage or detailed reason.
func (failure *SignedDocumentVerificationError) Error() string {
	return "signed document verification failed: " + string(failure.wireCode)
}

// WireCode returns the stable error safe to place on the protocol wire.
func (failure *SignedDocumentVerificationError) WireCode() WireErrorCode { return failure.wireCode }

// ProtectedDiagnostic returns local first-failure evidence that must not be relayed to peers.
func (failure *SignedDocumentVerificationError) ProtectedDiagnostic() VerificationDiagnostic {
	return failure.diagnostic
}

// Principal identifies the organization Principal bound to a resolved signing key.
type Principal struct {
	Type string
	ID   string
}

// SignatureEvidence is immutable retained signature-envelope evidence.
type SignatureEvidence struct {
	algorithm string
	createdAt string
	keyID     string
	value     string
	bytes     []byte
}

func (signature SignatureEvidence) clone() SignatureEvidence {
	signature.bytes = append([]byte(nil), signature.bytes...)
	return signature
}

// Algorithm returns the signature algorithm.
func (signature SignatureEvidence) Algorithm() string { return signature.algorithm }

// CreatedAt returns the exact signature.createdAt text received on the wire.
func (signature SignatureEvidence) CreatedAt() string { return signature.createdAt }

// KeyID returns the immutable Registry key identifier from the signature envelope.
func (signature SignatureEvidence) KeyID() string { return signature.keyID }

// Value returns the exact canonical base64url signature text received on the wire.
func (signature SignatureEvidence) Value() string { return signature.value }

// Bytes returns a copy of the decoded 64-byte Ed25519 signature.
func (signature SignatureEvidence) Bytes() []byte { return append([]byte(nil), signature.bytes...) }

// ResolvedKeyEvidence is immutable evidence selected from a fully validated Registry snapshot.
type ResolvedKeyEvidence struct {
	organizationID string
	keyID          string
	principal      Principal
	algorithm      string
	publicKeyText  string
	publicKeyBytes []byte
	validFrom      RFC3339Instant
	validUntil     *RFC3339Instant
	revokedAt      *RFC3339Instant
}

func (key ResolvedKeyEvidence) clone() ResolvedKeyEvidence {
	key.publicKeyBytes = append([]byte(nil), key.publicKeyBytes...)
	if key.validUntil != nil {
		value := *key.validUntil
		key.validUntil = &value
	}
	if key.revokedAt != nil {
		value := *key.revokedAt
		key.revokedAt = &value
	}
	return key
}

// OrganizationID returns the Registry organization identifier.
func (key ResolvedKeyEvidence) OrganizationID() string { return key.organizationID }

// KeyID returns the resolved immutable key identifier.
func (key ResolvedKeyEvidence) KeyID() string { return key.keyID }

// Principal returns the exact Principal bound to the key.
func (key ResolvedKeyEvidence) Principal() Principal { return key.principal }

// Algorithm returns the resolved key algorithm.
func (key ResolvedKeyEvidence) Algorithm() string { return key.algorithm }

// PublicKeyText returns the canonical base64url public-key spelling from the Registry.
func (key ResolvedKeyEvidence) PublicKeyText() string { return key.publicKeyText }

// PublicKeyBytes returns a copy of the decoded 32-byte public key.
func (key ResolvedKeyEvidence) PublicKeyBytes() []byte {
	return append([]byte(nil), key.publicKeyBytes...)
}

// ValidFrom returns the inclusive key-validity boundary.
func (key ResolvedKeyEvidence) ValidFrom() RFC3339Instant { return key.validFrom }

// ValidUntil returns the effective exclusive expiry boundary, if present.
func (key ResolvedKeyEvidence) ValidUntil() (RFC3339Instant, bool) {
	if key.validUntil == nil {
		return RFC3339Instant{}, false
	}
	return *key.validUntil, true
}

// RevokedAt returns the effective exclusive revocation boundary, if present.
func (key ResolvedKeyEvidence) RevokedAt() (RFC3339Instant, bool) {
	if key.revokedAt == nil {
		return RFC3339Instant{}, false
	}
	return *key.revokedAt, true
}

// VerifiedSignedDocument is immutable evidence produced only after all six verification stages.
type VerifiedSignedDocument struct {
	kind              SignedDocumentKind
	document          map[string]any
	receivedBytes     []byte
	signingBytes      []byte
	signingHash       string
	completeBytes     []byte
	completeHash      string
	protectedTime     string
	protectedInstant  RFC3339Instant
	signatureEvidence SignatureEvidence
	resolvedKey       ResolvedKeyEvidence
}

// Kind returns the explicit Signed Document profile used for verification.
func (verified *VerifiedSignedDocument) Kind() SignedDocumentKind { return verified.kind }

// Document returns a deep copy of the parsed signed document.
func (verified *VerifiedSignedDocument) Document() map[string]any {
	cloned, err := cloneJSONValue(verified.document)
	if err != nil {
		panic("verified JSON document became invalid: " + err.Error())
	}
	return cloned.(map[string]any)
}

// ReceivedBytes returns a copy of the exact UTF-8 bytes supplied to Verify.
func (verified *VerifiedSignedDocument) ReceivedBytes() []byte {
	return append([]byte(nil), verified.receivedBytes...)
}

// SigningBytes returns a copy of the RFC 8785 bytes covered by the Ed25519 signature.
func (verified *VerifiedSignedDocument) SigningBytes() []byte {
	return append([]byte(nil), verified.signingBytes...)
}

// SigningHash returns the lowercase sha256: identifier over SigningBytes.
func (verified *VerifiedSignedDocument) SigningHash() string { return verified.signingHash }

// CompleteBytes returns a copy of the complete signed document's RFC 8785 bytes.
func (verified *VerifiedSignedDocument) CompleteBytes() []byte {
	return append([]byte(nil), verified.completeBytes...)
}

// CompleteHash returns the lowercase sha256: identifier over CompleteBytes.
func (verified *VerifiedSignedDocument) CompleteHash() string { return verified.completeHash }

// ProtectedTime returns the exact protected signed-time text received on the wire.
func (verified *VerifiedSignedDocument) ProtectedTime() string { return verified.protectedTime }

// ProtectedInstant returns the parsed protected instant without fractional truncation.
func (verified *VerifiedSignedDocument) ProtectedInstant() RFC3339Instant {
	return verified.protectedInstant
}

// Signature returns an immutable copy of retained signature material.
func (verified *VerifiedSignedDocument) Signature() SignatureEvidence {
	return verified.signatureEvidence.clone()
}

// ResolvedKey returns an immutable copy of resolved Registry evidence.
func (verified *VerifiedSignedDocument) ResolvedKey() ResolvedKeyEvidence {
	return verified.resolvedKey.clone()
}

type envelopeResult struct {
	protectedTime    string
	protectedInstant RFC3339Instant
	signature        SignatureEvidence
	exactPrincipal   *Principal
	servicePrincipal bool
}

// Verify performs the six ordered Signed Document verification stages for one explicit profile.
func (codec *SignedDocumentCodec) Verify(
	kind SignedDocumentKind,
	raw []byte,
	resolver KeyResolver,
) (*VerifiedSignedDocument, error) {
	profile, ok := signedDocumentProfiles[kind]
	if !ok {
		return nil, fmt.Errorf("unsupported signed document kind %q", kind)
	}
	if resolver == nil {
		return nil, errors.New("KeyResolver must not be nil")
	}
	received := append([]byte(nil), raw...)
	hasInvalidSurrogate := containsUnpairedSurrogateEscape(raw)
	value, err := DecodeJSON(raw)
	if err != nil {
		return nil, verificationFailure(VerificationParse, err.Error())
	}
	if err := codec.catalog.validateValue(profile.schemaName, value); err != nil {
		return nil, verificationFailure(VerificationSchema, err.Error())
	}
	document, ok := value.(map[string]any)
	if !ok {
		return nil, verificationFailure(VerificationSchema, "signed protocol document is not an object")
	}
	envelope, failure := verifySignatureEnvelope(document, profile)
	if failure != nil {
		return nil, failure
	}

	expectedSigner := ExpectedSigner{Rule: ExpectedSignerServicePrincipal}
	if envelope.exactPrincipal != nil {
		expectedSigner = ExpectedSigner{
			Rule:      ExpectedSignerExactPrincipal,
			Principal: *envelope.exactPrincipal,
		}
	}
	snapshot, err := resolver.Resolve(KeyResolutionRequest{
		Kind:           kind,
		KeyID:          envelope.signature.keyID,
		ExpectedSigner: expectedSigner,
		ProtectedTime: ProtectedSignedTime{
			Text:    envelope.protectedTime,
			Instant: envelope.protectedInstant,
		},
	})
	if err != nil {
		return nil, verificationFailure(VerificationKeyResolution, "KeyResolver failed: "+err.Error())
	}
	if snapshot.Completeness != KeyRegistryOrganizationWide {
		return nil, verificationFailure(
			VerificationKeyResolution,
			"KeyResolver did not establish organization-wide Registry completeness",
		)
	}
	key, failure := resolveRegistryKey(append([]byte(nil), snapshot.RegistryBytes...), envelope)
	if failure != nil {
		return nil, failure
	}

	if hasInvalidSurrogate {
		return nil, verificationFailure(
			VerificationCanonicalization,
			"document contains an unpaired Unicode surrogate escape",
		)
	}
	unsigned := make(map[string]any, len(document)-1)
	for name, item := range document {
		if name != "signature" {
			unsigned[name] = item
		}
	}
	signingBytes, err := MarshalCanonicalJSON(unsigned)
	if err != nil {
		return nil, verificationFailure(VerificationCanonicalization, err.Error())
	}
	if !ed25519.Verify(ed25519.PublicKey(key.publicKeyBytes), signingBytes, envelope.signature.bytes) {
		return nil, verificationFailure(VerificationSignature, "Ed25519 signature does not verify")
	}
	completeBytes, err := MarshalCanonicalJSON(document)
	if err != nil {
		return nil, verificationFailure(VerificationCanonicalization, err.Error())
	}
	return &VerifiedSignedDocument{
		kind:              kind,
		document:          document,
		receivedBytes:     received,
		signingBytes:      append([]byte(nil), signingBytes...),
		signingHash:       sha256Identifier(signingBytes),
		completeBytes:     append([]byte(nil), completeBytes...),
		completeHash:      sha256Identifier(completeBytes),
		protectedTime:     envelope.protectedTime,
		protectedInstant:  envelope.protectedInstant,
		signatureEvidence: envelope.signature.clone(),
		resolvedKey:       key.clone(),
	}, nil
}

func verificationFailure(stage VerificationStage, reason string) *SignedDocumentVerificationError {
	code := WireAuthInvalidSignature
	switch stage {
	case VerificationParse, VerificationCanonicalization:
		code = WireProtocolViolation
	case VerificationSchema:
		code = WireSchemaValidationFailed
	}
	return &SignedDocumentVerificationError{
		wireCode:   code,
		diagnostic: VerificationDiagnostic{stage: stage, reason: reason},
	}
}

func verifySignatureEnvelope(
	document map[string]any,
	profile signedDocumentProfile,
) (envelopeResult, *SignedDocumentVerificationError) {
	signatureObject, ok := document["signature"].(map[string]any)
	if !ok {
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "signature is not an object")
	}
	algorithm, algorithmOK := signatureObject["algorithm"].(string)
	createdAt, createdAtOK := signatureObject["createdAt"].(string)
	keyID, keyIDOK := signatureObject["keyId"].(string)
	value, valueOK := signatureObject["value"].(string)
	if !algorithmOK || !createdAtOK || !keyIDOK || !valueOK || algorithm != "Ed25519" {
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "signature envelope fields are malformed")
	}
	protectedTime, ok := jsonPointer(document, profile.protectedTimePointer).(string)
	if !ok {
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "protected signed time is not a string")
	}
	protectedInstant, err := parseProtocolRFC3339(protectedTime)
	if err != nil {
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "protected signed time is invalid: "+err.Error())
	}
	if _, err := parseProtocolRFC3339(createdAt); err != nil {
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "signature.createdAt is invalid: "+err.Error())
	}
	if !strings.HasSuffix(protectedTime, "Z") || !strings.HasSuffix(createdAt, "Z") {
		return envelopeResult{}, verificationFailure(
			VerificationSignatureEnvelope,
			"protected time and signature.createdAt must use uppercase Z",
		)
	}
	if protectedTime != createdAt {
		return envelopeResult{}, verificationFailure(
			VerificationSignatureEnvelope,
			"protected time and signature.createdAt are not byte-equal",
		)
	}

	envelope := envelopeResult{protectedTime: protectedTime, protectedInstant: protectedInstant}
	switch profile.expectedSignerRule {
	case expectPrincipalObject:
		principal, err := principalFromValue(jsonPointer(document, profile.expectedSignerPointer))
		if err != nil {
			return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "expected signer is not a Principal object")
		}
		envelope.exactPrincipal = &principal
	case expectAgentID:
		agentID, ok := jsonPointer(document, profile.expectedSignerPointer).(string)
		if !ok {
			return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "expected Agent signer ID is not a string")
		}
		principal := Principal{Type: "agent", ID: agentID}
		envelope.exactPrincipal = &principal
	case expectServicePrincipal:
		envelope.servicePrincipal = true
	default:
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "unsupported expected signer rule")
	}

	signatureBytes, err := canonicalBase64URL(value)
	if err != nil {
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "signature.value "+err.Error())
	}
	if len(signatureBytes) != ed25519.SignatureSize {
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "signature.value does not decode to 64 bytes")
	}
	if err := strictEd25519Point(signatureBytes[:32], true); err != nil {
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "signature R "+err.Error())
	}
	if littleEndianInteger(signatureBytes[32:]).Cmp(ed25519Order) >= 0 {
		return envelopeResult{}, verificationFailure(VerificationSignatureEnvelope, "signature S is outside the Ed25519 scalar range")
	}
	envelope.signature = SignatureEvidence{
		algorithm: algorithm,
		createdAt: createdAt,
		keyID:     keyID,
		value:     value,
		bytes:     signatureBytes,
	}
	return envelope, nil
}

type registryStatus struct {
	sequence   int64
	recordedAt RFC3339Instant
	validUntil *RFC3339Instant
	revokedAt  *RFC3339Instant
}

type normalizedRegistryBinding struct {
	principal      Principal
	algorithm      string
	publicKeyText  string
	publicKeyBytes []byte
	validFrom      RFC3339Instant
	history        map[int64]registryStatus
	validUntil     *RFC3339Instant
	revokedAt      *RFC3339Instant
}

func resolveRegistryKey(
	raw []byte,
	envelope envelopeResult,
) (ResolvedKeyEvidence, *SignedDocumentVerificationError) {
	value, err := DecodeJSON(raw)
	if err != nil {
		return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, err.Error())
	}
	registry, err := exactJSONObject(value, []string{"organizationId", "bindings"}, nil, "Registry")
	if err != nil {
		return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, err.Error())
	}
	organizationID, ok := registry["organizationId"].(string)
	if !ok {
		return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, "Registry organizationId is not a string")
	}
	bindings, ok := registry["bindings"].([]any)
	if !ok || len(bindings) == 0 {
		return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, "Registry bindings is not a non-empty array")
	}

	normalized := make(map[string]*normalizedRegistryBinding)
	type owner struct{ keyID, principalType, principalID string }
	publicKeyOwners := make(map[string]owner)
	type tuple struct{ principalType, principalID, algorithm, publicKey string }
	tupleIDs := make(map[tuple]string)
	for index, rawBinding := range bindings {
		label := fmt.Sprintf("Registry bindings[%d]", index)
		binding, err := exactJSONObject(rawBinding, []string{
			"keyId", "principal", "algorithm", "publicKey", "validFrom", "validityHistory",
		}, nil, label)
		if err != nil {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, err.Error())
		}
		keyID, ok := binding["keyId"].(string)
		if !ok {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".keyId is not a string")
		}
		principal, err := principalFromValue(binding["principal"])
		if err != nil {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".principal "+err.Error())
		}
		algorithm, ok := binding["algorithm"].(string)
		if !ok || algorithm != "Ed25519" {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".algorithm is not Ed25519")
		}
		publicKeyText, ok := binding["publicKey"].(string)
		if !ok {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".publicKey is not a string")
		}
		publicKeyBytes, err := canonicalBase64URL(publicKeyText)
		if err != nil {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".publicKey "+err.Error())
		}
		if len(publicKeyBytes) != ed25519.PublicKeySize {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".publicKey does not decode to 32 bytes")
		}
		if err := strictEd25519Point(publicKeyBytes, false); err != nil {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".publicKey "+err.Error())
		}
		validFromText, ok := binding["validFrom"].(string)
		if !ok {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".validFrom is not a string")
		}
		validFrom, err := parseProtocolRFC3339(validFromText)
		if err != nil {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".validFrom is invalid: "+err.Error())
		}
		history, ok := binding["validityHistory"].([]any)
		if !ok {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, label+".validityHistory is not an array")
		}

		existing := normalized[keyID]
		if existing != nil {
			if existing.principal != principal || existing.algorithm != algorithm ||
				string(existing.publicKeyBytes) != string(publicKeyBytes) || existing.validFrom.compare(validFrom) != 0 {
				return ResolvedKeyEvidence{}, verificationFailure(
					VerificationKeyResolution,
					fmt.Sprintf("key ID %q is reused for another immutable binding", keyID),
				)
			}
		} else {
			existing = &normalizedRegistryBinding{
				principal: principal, algorithm: algorithm,
				publicKeyText: publicKeyText, publicKeyBytes: append([]byte(nil), publicKeyBytes...),
				validFrom: validFrom, history: make(map[int64]registryStatus),
			}
			normalized[keyID] = existing
		}

		ownerValue := owner{keyID: keyID, principalType: principal.Type, principalID: principal.ID}
		if previous, exists := publicKeyOwners[string(publicKeyBytes)]; exists && previous != ownerValue {
			return ResolvedKeyEvidence{}, verificationFailure(
				VerificationKeyResolution,
				"the same public key is registered under another Principal or key ID",
			)
		}
		publicKeyOwners[string(publicKeyBytes)] = ownerValue
		tupleValue := tuple{principal.Type, principal.ID, algorithm, string(publicKeyBytes)}
		if alias, exists := tupleIDs[tupleValue]; exists && alias != keyID {
			return ResolvedKeyEvidence{}, verificationFailure(
				VerificationKeyResolution,
				"a Principal, algorithm, and public-key tuple has a key-ID alias",
			)
		}
		tupleIDs[tupleValue] = keyID

		for statusIndex, rawStatus := range history {
			statusLabel := fmt.Sprintf("%s.validityHistory[%d]", label, statusIndex)
			statusObject, err := exactJSONObject(
				rawStatus,
				[]string{"sequence", "recordedAt"},
				[]string{"validUntil", "revokedAt"},
				statusLabel,
			)
			if err != nil {
				return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, err.Error())
			}
			sequence, err := positiveSafeInteger(statusObject["sequence"])
			if err != nil {
				return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, statusLabel+".sequence "+err.Error())
			}
			recordedAt, err := requiredInstant(statusObject, "recordedAt", statusLabel)
			if err != nil {
				return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, err.Error())
			}
			status := registryStatus{sequence: sequence, recordedAt: recordedAt}
			if _, exists := statusObject["validUntil"]; exists {
				instant, err := requiredInstant(statusObject, "validUntil", statusLabel)
				if err != nil {
					return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, err.Error())
				}
				status.validUntil = &instant
			}
			if _, exists := statusObject["revokedAt"]; exists {
				instant, err := requiredInstant(statusObject, "revokedAt", statusLabel)
				if err != nil {
					return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, err.Error())
				}
				status.revokedAt = &instant
			}
			if previous, exists := existing.history[sequence]; exists && !registryStatusesEqual(previous, status) {
				return ResolvedKeyEvidence{}, verificationFailure(
					VerificationKeyResolution,
					statusLabel+" rewrites an earlier status sequence",
				)
			}
			existing.history[sequence] = status
		}
	}

	for keyID, binding := range normalized {
		var recordedAt *RFC3339Instant
		for sequence := int64(1); sequence <= int64(len(binding.history)); sequence++ {
			status, exists := binding.history[sequence]
			if !exists {
				return ResolvedKeyEvidence{}, verificationFailure(
					VerificationKeyResolution,
					fmt.Sprintf("key %q validity history is not contiguous from sequence 1", keyID),
				)
			}
			if recordedAt != nil && status.recordedAt.compare(*recordedAt) < 0 {
				return ResolvedKeyEvidence{}, verificationFailure(
					VerificationKeyResolution,
					fmt.Sprintf("key %q validity history is not append ordered", keyID),
				)
			}
			instant := status.recordedAt
			recordedAt = &instant
			if status.validUntil != nil {
				if binding.validUntil != nil && status.validUntil.compare(*binding.validUntil) > 0 {
					return ResolvedKeyEvidence{}, verificationFailure(
						VerificationKeyResolution,
						fmt.Sprintf("key %q moves validUntil later in history", keyID),
					)
				}
				value := *status.validUntil
				binding.validUntil = &value
			}
			if status.revokedAt != nil {
				if binding.revokedAt != nil && status.revokedAt.compare(*binding.revokedAt) > 0 {
					return ResolvedKeyEvidence{}, verificationFailure(
						VerificationKeyResolution,
						fmt.Sprintf("key %q moves revokedAt later in history", keyID),
					)
				}
				value := *status.revokedAt
				binding.revokedAt = &value
			}
		}
	}

	selected := normalized[envelope.signature.keyID]
	if selected == nil {
		return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, "signature.keyId is unknown")
	}
	if envelope.servicePrincipal {
		if selected.principal.Type != "service" {
			return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, "Agent Card signer is not a service Principal")
		}
	} else if envelope.exactPrincipal == nil || selected.principal != *envelope.exactPrincipal {
		return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, "resolved key is bound to the wrong Principal")
	}
	protected := envelope.protectedInstant
	if protected.compare(selected.validFrom) < 0 {
		return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, "signing key is not yet valid at the protected time")
	}
	if selected.validUntil != nil && protected.compare(*selected.validUntil) >= 0 {
		return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, "signing key is expired at the protected time")
	}
	if selected.revokedAt != nil && protected.compare(*selected.revokedAt) >= 0 {
		return ResolvedKeyEvidence{}, verificationFailure(VerificationKeyResolution, "signing key is revoked at the protected time")
	}
	return ResolvedKeyEvidence{
		organizationID: organizationID,
		keyID:          envelope.signature.keyID,
		principal:      selected.principal,
		algorithm:      selected.algorithm,
		publicKeyText:  selected.publicKeyText,
		publicKeyBytes: append([]byte(nil), selected.publicKeyBytes...),
		validFrom:      selected.validFrom,
		validUntil:     selected.validUntil,
		revokedAt:      selected.revokedAt,
	}, nil
}

func exactJSONObject(value any, required, optional []string, label string) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s is not an object", label)
	}
	allowed := make(map[string]bool, len(required)+len(optional))
	for _, name := range required {
		allowed[name] = true
		if _, exists := object[name]; !exists {
			return nil, fmt.Errorf("%s is missing %s", label, name)
		}
	}
	for _, name := range optional {
		allowed[name] = true
	}
	for name := range object {
		if !allowed[name] {
			return nil, fmt.Errorf("%s has unknown field %s", label, name)
		}
	}
	return object, nil
}

func principalFromValue(value any) (Principal, error) {
	object, err := exactJSONObject(value, []string{"type", "id"}, nil, "Principal")
	if err != nil {
		return Principal{}, err
	}
	principalType, typeOK := object["type"].(string)
	principalID, idOK := object["id"].(string)
	if !typeOK || !idOK {
		return Principal{}, errors.New("fields are not strings")
	}
	if principalType != "agent" && principalType != "human" && principalType != "service" {
		return Principal{}, errors.New("type is unsupported")
	}
	return Principal{Type: principalType, ID: principalID}, nil
}

func canonicalBase64URL(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("is not unpadded base64url")
	}
	for _, character := range value {
		if !(character >= 'A' && character <= 'Z') &&
			!(character >= 'a' && character <= 'z') &&
			!(character >= '0' && character <= '9') && character != '_' && character != '-' {
			return nil, errors.New("is not unpadded base64url")
		}
	}
	if len(value)%4 == 1 {
		return nil, errors.New("has an impossible base64url length")
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil {
		return nil, errors.New("cannot be decoded as canonical base64url")
	}
	if base64.RawURLEncoding.EncodeToString(decoded) != value {
		return nil, errors.New("has nonzero unused pad bits or noncanonical spelling")
	}
	return decoded, nil
}

func positiveSafeInteger(value any) (int64, error) {
	const maximum = int64(9007199254740991)
	switch number := value.(type) {
	case json.Number:
		rational, ok := new(big.Rat).SetString(string(number))
		if !ok || !rational.IsInt() || rational.Sign() <= 0 || rational.Num().BitLen() > 53 {
			return 0, errors.New("is outside the positive safe-integer range")
		}
		integer := rational.Num().Int64()
		if integer > maximum {
			return 0, errors.New("is outside the positive safe-integer range")
		}
		return integer, nil
	case float64:
		if math.IsNaN(number) || math.IsInf(number, 0) || number != math.Trunc(number) || number < 1 || number > float64(maximum) {
			return 0, errors.New("is outside the positive safe-integer range")
		}
		return int64(number), nil
	default:
		return 0, errors.New("is not an integer")
	}
}

func requiredInstant(object map[string]any, field, label string) (RFC3339Instant, error) {
	text, ok := object[field].(string)
	if !ok {
		return RFC3339Instant{}, fmt.Errorf("%s.%s is not a string", label, field)
	}
	instant, err := parseProtocolRFC3339(text)
	if err != nil {
		return RFC3339Instant{}, fmt.Errorf("%s.%s is invalid: %w", label, field, err)
	}
	return instant, nil
}

func registryStatusesEqual(left, right registryStatus) bool {
	if left.sequence != right.sequence || left.recordedAt.compare(right.recordedAt) != 0 {
		return false
	}
	return optionalInstantsEqual(left.validUntil, right.validUntil) &&
		optionalInstantsEqual(left.revokedAt, right.revokedAt)
}

func optionalInstantsEqual(left, right *RFC3339Instant) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.compare(*right) == 0
}

func containsUnpairedSurrogateEscape(raw []byte) bool {
	inString := false
	for index := 0; index < len(raw); index++ {
		if !inString {
			if raw[index] == '"' {
				inString = true
			}
			continue
		}
		switch raw[index] {
		case '"':
			inString = false
		case '\\':
			if index+1 >= len(raw) {
				continue
			}
			if raw[index+1] != 'u' || index+5 >= len(raw) {
				index++
				continue
			}
			code, ok := decodeHexQuad(raw[index+2 : index+6])
			if !ok {
				index += 5
				continue
			}
			if code >= 0xd800 && code <= 0xdbff {
				if index+11 >= len(raw) || raw[index+6] != '\\' || raw[index+7] != 'u' {
					return true
				}
				low, ok := decodeHexQuad(raw[index+8 : index+12])
				if !ok || low < 0xdc00 || low > 0xdfff {
					return true
				}
				index += 11
				continue
			}
			if code >= 0xdc00 && code <= 0xdfff {
				return true
			}
			index += 5
		}
	}
	return false
}

func decodeHexQuad(value []byte) (uint16, bool) {
	if len(value) != 4 {
		return 0, false
	}
	var result uint16
	for _, character := range value {
		result <<= 4
		switch {
		case character >= '0' && character <= '9':
			result |= uint16(character - '0')
		case character >= 'a' && character <= 'f':
			result |= uint16(character-'a') + 10
		case character >= 'A' && character <= 'F':
			result |= uint16(character-'A') + 10
		default:
			return 0, false
		}
	}
	return result, true
}
