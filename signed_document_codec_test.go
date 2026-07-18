package missionweaveprotocol_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

type fixtureSigningKey struct {
	keyID      string
	privateKey ed25519.PrivateKey
}

type recordingSigningKey struct {
	calls     int
	signature []byte
}

func (key *recordingSigningKey) Algorithm() string { return "Ed25519" }

func (key *recordingSigningKey) KeyID() string {
	return "urn:missionweaveprotocol:key:crypto-vector-rfc8032-1"
}

func (key *recordingSigningKey) Sign(_ []byte) ([]byte, error) {
	key.calls++
	return append([]byte(nil), key.signature...), nil
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

func (key fixtureSigningKey) Algorithm() string { return "Ed25519" }

func (key fixtureSigningKey) KeyID() string { return key.keyID }

func (key fixtureSigningKey) Sign(message []byte) ([]byte, error) {
	return ed25519.Sign(key.privateKey, message), nil
}

func TestSignedDocumentCodecSignsGoldenCommand(t *testing.T) {
	expected := readJSONObject(t, "cryptography/vectors/signed-documents/valid/command.json")
	unsigned := readJSONObject(t, "cryptography/vectors/signed-documents/valid/command.json")
	delete(unsigned, "signature")
	fixture := readJSONObject(t, "cryptography/keys/signing-coordinator.json")
	seed, err := base64.RawURLEncoding.Strict().DecodeString(fixture["seed"].(string))
	if err != nil {
		t.Fatal(err)
	}

	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}
	signed, err := codec.Sign(
		missionweaveprotocol.SignedDocumentCommand,
		unsigned,
		fixtureSigningKey{
			keyID:      fixture["keyId"].(string),
			privateKey: ed25519.NewKeyFromSeed(seed),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(signed, expected) {
		t.Fatalf("signed Command differs from the protocol golden vector\nactual: %#v\nexpected: %#v", signed, expected)
	}
	if _, mutated := unsigned["signature"]; mutated {
		t.Fatal("Sign mutated the caller's unsigned document")
	}
}

func TestSignedDocumentCodecVerifiesGoldenCommandAndRetainsEvidence(t *testing.T) {
	raw, err := missionweaveprotocol.ReadProtocolFile(
		"cryptography/vectors/signed-documents/valid/command.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := missionweaveprotocol.ReadProtocolFile("cryptography/keys/registry-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	expectedSigningBytes, err := missionweaveprotocol.ReadProtocolFile(
		"cryptography/vectors/canonicalization/command.signing.jcs",
	)
	if err != nil {
		t.Fatal(err)
	}
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}
	verified, err := codec.Verify(
		missionweaveprotocol.SignedDocumentCommand,
		raw,
		fixtureKeyResolver{registry: registry},
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(verified.Document(), readJSONObject(t, "cryptography/vectors/signed-documents/valid/command.json")) {
		t.Fatal("verified document differs from received JSON")
	}
	if string(verified.ReceivedBytes()) != string(raw) {
		t.Fatal("verified result did not retain exact received bytes")
	}
	if string(verified.SigningBytes()) != string(expectedSigningBytes) {
		t.Fatal("verified result retained the wrong signing bytes")
	}
	if verified.SigningHash() != "sha256:6655c5d67ae3ecc19a4ed04bda7f1372aeaafc7adf939a77715de96ef2100695" {
		t.Fatalf("unexpected signing hash: %s", verified.SigningHash())
	}
	if verified.CompleteHash() != "sha256:1d17d0bd5379e554d48d14a6b328671f12860c6c3278bc1e7ca4e1163a74353f" {
		t.Fatalf("unexpected complete hash: %s", verified.CompleteHash())
	}
	if verified.ProtectedTime() != "2026-07-15T00:00:00Z" || verified.ProtectedInstant().Fraction() != "" {
		t.Fatal("verified result did not retain both protected-time forms")
	}
	signature := verified.Signature()
	if signature.Algorithm() != "Ed25519" ||
		signature.KeyID() != "urn:missionweaveprotocol:key:crypto-vector-rfc8032-1" ||
		signature.Value() != "PMeeKgpw-HlGNwHbQbEMrfAxbw1815fBdFhOSTHy31ss90eTcuQ4rWeRZbmqFFtHgLKzd0gNm67-HenzwGVhAg" {
		t.Fatal("verified result retained the wrong signature material")
	}
	key := verified.ResolvedKey()
	if key.KeyID() != signature.KeyID() || key.Principal() != (missionweaveprotocol.Principal{
		Type: "agent",
		ID:   "urn:missionweaveprotocol:agent:crypto-vector-coordinator",
	}) {
		t.Fatal("verified result retained the wrong resolved Principal evidence")
	}

	receivedCopy := verified.ReceivedBytes()
	receivedCopy[0] ^= 0xff
	signingCopy := verified.SigningBytes()
	signingCopy[0] ^= 0xff
	documentCopy := verified.Document()
	documentCopy["issuedAt"] = "mutated"
	if verified.ReceivedBytes()[0] != raw[0] ||
		verified.SigningBytes()[0] != expectedSigningBytes[0] ||
		verified.Document()["issuedAt"] != "2026-07-15T00:00:00Z" {
		t.Fatal("verified result exposed mutable retained evidence")
	}
}

func TestSignedDocumentCodecReportsFirstFailureWithoutAnOracle(t *testing.T) {
	registry, err := missionweaveprotocol.ReadProtocolFile("cryptography/keys/registry-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		document string
		stage    missionweaveprotocol.VerificationStage
		wire     missionweaveprotocol.WireErrorCode
	}{
		{
			name: "parse", document: "cryptography/vectors/signed-documents/invalid/command-invalid-utf8.bin",
			stage: missionweaveprotocol.VerificationParse, wire: missionweaveprotocol.WireProtocolViolation,
		},
		{
			name: "schema", document: "cryptography/vectors/signed-documents/invalid/command-unsupported-algorithm.json",
			stage: missionweaveprotocol.VerificationSchema, wire: missionweaveprotocol.WireSchemaValidationFailed,
		},
		{
			name: "signature envelope", document: "cryptography/vectors/signed-documents/invalid/command-created-at-mismatch.json",
			stage: missionweaveprotocol.VerificationSignatureEnvelope, wire: missionweaveprotocol.WireAuthInvalidSignature,
		},
		{
			name: "key resolution", document: "cryptography/vectors/signed-documents/invalid/command-unknown-key.json",
			stage: missionweaveprotocol.VerificationKeyResolution, wire: missionweaveprotocol.WireAuthInvalidSignature,
		},
		{
			name: "canonicalization", document: "cryptography/vectors/signed-documents/invalid/command-number-1e400.json",
			stage: missionweaveprotocol.VerificationCanonicalization, wire: missionweaveprotocol.WireProtocolViolation,
		},
		{
			name: "signature", document: "cryptography/vectors/signed-documents/invalid/command-payload-tamper.json",
			stage: missionweaveprotocol.VerificationSignature, wire: missionweaveprotocol.WireAuthInvalidSignature,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw, err := missionweaveprotocol.ReadProtocolFile(test.document)
			if err != nil {
				t.Fatal(err)
			}
			_, err = codec.Verify(
				missionweaveprotocol.SignedDocumentCommand,
				raw,
				fixtureKeyResolver{registry: registry},
			)
			var failure *missionweaveprotocol.SignedDocumentVerificationError
			if !errors.As(err, &failure) {
				t.Fatalf("expected SignedDocumentVerificationError, got %T: %v", err, err)
			}
			if failure.WireCode() != test.wire {
				t.Fatalf("wire code = %s; want %s", failure.WireCode(), test.wire)
			}
			diagnostic := failure.ProtectedDiagnostic()
			if diagnostic.Stage() != test.stage || diagnostic.Reason() == "" {
				t.Fatalf("diagnostic = (%s, %q); want stage %s and a reason", diagnostic.Stage(), diagnostic.Reason(), test.stage)
			}
			if failure.Error() != "signed document verification failed: "+string(test.wire) {
				t.Fatalf("public error leaked protected diagnostics: %q", failure.Error())
			}
		})
	}
}

func TestSignedDocumentCodecRejectsCoerciveSigningInput(t *testing.T) {
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}
	key := fixtureSigningKey{
		keyID:      "urn:missionweaveprotocol:key:test",
		privateKey: ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)),
	}
	_, err = codec.Sign(
		missionweaveprotocol.SignedDocumentCommand,
		map[string]any{"issuedAt": "2026-07-15T00:00:00Z", "coercive": 1},
		key,
	)
	if err == nil {
		t.Fatal("Sign accepted a non-JSON-domain Go int")
	}
	_, err = codec.Sign(
		missionweaveprotocol.SignedDocumentCommand,
		map[string]any{"issuedAt": "2026-07-15T00:00:00Z", "signature": nil},
		key,
	)
	if err == nil {
		t.Fatal("Sign accepted an existing top-level signature")
	}
}

func TestSignedDocumentCodecFailsClosedOnIncompleteRegistrySnapshot(t *testing.T) {
	raw, err := missionweaveprotocol.ReadProtocolFile(
		"cryptography/vectors/signed-documents/valid/command.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := missionweaveprotocol.ReadProtocolFile("cryptography/keys/registry-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}
	_, err = codec.Verify(
		missionweaveprotocol.SignedDocumentCommand,
		raw,
		incompleteKeyResolver{registry: registry},
	)
	var failure *missionweaveprotocol.SignedDocumentVerificationError
	if !errors.As(err, &failure) {
		t.Fatalf("expected SignedDocumentVerificationError, got %T: %v", err, err)
	}
	if failure.ProtectedDiagnostic().Stage() != missionweaveprotocol.VerificationKeyResolution ||
		failure.WireCode() != missionweaveprotocol.WireAuthInvalidSignature {
		t.Fatalf("incomplete Registry failed as %s/%s", failure.ProtectedDiagnostic().Stage(), failure.WireCode())
	}
}

func TestSignedDocumentCodecValidatesProtectedTimeBeforeCallingSigningKey(t *testing.T) {
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}
	unsigned := readJSONObject(t, "cryptography/vectors/signed-documents/valid/command.json")
	delete(unsigned, "signature")
	unsigned["issuedAt"] = "2026-07-15T00:00:00+00:00"
	key := &recordingSigningKey{signature: make([]byte, ed25519.SignatureSize)}
	if _, err := codec.Sign(missionweaveprotocol.SignedDocumentCommand, unsigned, key); err == nil {
		t.Fatal("Sign accepted a protected time without uppercase Z")
	}
	if key.calls != 0 {
		t.Fatal("Sign called SigningKey before validating the protected time")
	}
}

func TestSignedDocumentCodecRejectsMalformedSignatureReturnedByAdapter(t *testing.T) {
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		fixture string
	}{
		{
			name:    "R outside prime-order subgroup",
			fixture: "cryptography/vectors/signed-documents/invalid/command-signature-r-small-order.json",
		},
		{
			name:    "S outside scalar range",
			fixture: "cryptography/vectors/signed-documents/invalid/command-signature-s-out-of-range.json",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			unsigned := readJSONObject(t, "cryptography/vectors/signed-documents/valid/command.json")
			delete(unsigned, "signature")
			invalid := readJSONObject(t, test.fixture)
			signatureObject := invalid["signature"].(map[string]any)
			signature, err := base64.RawURLEncoding.Strict().DecodeString(signatureObject["value"].(string))
			if err != nil {
				t.Fatal(err)
			}
			key := &recordingSigningKey{signature: signature}
			if _, err := codec.Sign(missionweaveprotocol.SignedDocumentCommand, unsigned, key); err == nil {
				t.Fatal("Sign accepted malformed Ed25519 signature material")
			}
			if key.calls != 1 {
				t.Fatalf("SigningKey calls = %d; want 1", key.calls)
			}
		})
	}
}

type incompleteKeyResolver struct {
	registry []byte
}

func (resolver incompleteKeyResolver) Resolve(
	_ missionweaveprotocol.KeyResolutionRequest,
) (missionweaveprotocol.KeyRegistrySnapshot, error) {
	return missionweaveprotocol.KeyRegistrySnapshot{RegistryBytes: resolver.registry}, nil
}

type capturingKeyResolver struct {
	registry []byte
	request  missionweaveprotocol.KeyResolutionRequest
}

func (resolver *capturingKeyResolver) Resolve(
	request missionweaveprotocol.KeyResolutionRequest,
) (missionweaveprotocol.KeyRegistrySnapshot, error) {
	resolver.request = request
	return missionweaveprotocol.KeyRegistrySnapshot{
		Completeness:  missionweaveprotocol.KeyRegistryOrganizationWide,
		RegistryBytes: append([]byte(nil), resolver.registry...),
	}, nil
}

func TestSignedDocumentCodecPassesResolutionContextToResolver(t *testing.T) {
	raw, err := missionweaveprotocol.ReadProtocolFile(
		"cryptography/vectors/signed-documents/valid/command.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := missionweaveprotocol.ReadProtocolFile("cryptography/keys/registry-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	resolver := &capturingKeyResolver{registry: registry}
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.Verify(missionweaveprotocol.SignedDocumentCommand, raw, resolver); err != nil {
		t.Fatal(err)
	}
	request := resolver.request
	if request.Kind != missionweaveprotocol.SignedDocumentCommand ||
		request.KeyID != "urn:missionweaveprotocol:key:crypto-vector-rfc8032-1" ||
		request.ExpectedSigner.Rule != missionweaveprotocol.ExpectedSignerExactPrincipal ||
		request.ExpectedSigner.Principal != (missionweaveprotocol.Principal{
			Type: "agent",
			ID:   "urn:missionweaveprotocol:agent:crypto-vector-coordinator",
		}) || request.ProtectedTime.Text != "2026-07-15T00:00:00Z" {
		t.Fatalf("unexpected key resolution request: %#v", request)
	}
}

func TestSignedDocumentCodecDoesNotApplyFixtureOnlyRegistrySizeLimit(t *testing.T) {
	raw, err := missionweaveprotocol.ReadProtocolFile(
		"cryptography/vectors/signed-documents/valid/command.json",
	)
	if err != nil {
		t.Fatal(err)
	}
	registry := readJSONObject(t, "cryptography/keys/registry-valid.json")
	bindings := registry["bindings"].([]any)
	large := make([]any, 65)
	for index := range large {
		large[index] = bindings[0]
	}
	registry["bindings"] = large
	registryBytes, err := json.Marshal(registry)
	if err != nil {
		t.Fatal(err)
	}
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := codec.Verify(
		missionweaveprotocol.SignedDocumentCommand,
		raw,
		fixtureKeyResolver{registry: registryBytes},
	); err != nil {
		t.Fatalf("organization Registry with 65 bindings was rejected: %v", err)
	}
}

func readJSONObject(t *testing.T, name string) map[string]any {
	t.Helper()
	document, err := missionweaveprotocol.ReadProtocolFile(name)
	if err != nil {
		t.Fatal(err)
	}
	value, err := missionweaveprotocol.DecodeJSON(document)
	if err != nil {
		t.Fatal(err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s is not a JSON object", name)
	}
	return object
}
