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

type cryptographyManifest struct {
	Cases []cryptographyCase `json:"cases"`
}

type cryptographyCase struct {
	ID          string                   `json:"id"`
	Kind        string                   `json:"kind"`
	Evaluations []cryptographyEvaluation `json:"evaluations"`
}

type cryptographyEvaluation struct {
	ProfileID   string                  `json:"profileId"`
	Document    string                  `json:"document"`
	Registry    string                  `json:"registry"`
	SigningKey  string                  `json:"signingKey"`
	Input       string                  `json:"input"`
	ExpectedJCS string                  `json:"expectedJcs"`
	SHA256      string                  `json:"sha256"`
	Expect      cryptographyExpectation `json:"expect"`
}

type cryptographyExpectation struct {
	Stage    string                        `json:"stage"`
	WireCode *string                       `json:"wireCode"`
	Verified *cryptographyVerifiedEvidence `json:"verified"`
}

type cryptographyVerifiedEvidence struct {
	KeyID              string            `json:"keyId"`
	Principal          manifestPrincipal `json:"principal"`
	ProtectedTime      string            `json:"protectedTime"`
	SigningBytes       string            `json:"signingBytes"`
	SigningHash        string            `json:"signingHash"`
	Signature          string            `json:"signature"`
	SignedDocumentHash string            `json:"signedDocumentHash"`
}

type manifestPrincipal struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func TestSignedDocumentCodecPassesAllCryptographyEvaluations(t *testing.T) {
	manifestBytes, err := missionweaveprotocol.ReadProtocolFile("cryptography/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest cryptographyManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	codec, err := missionweaveprotocol.NewSignedDocumentCodec()
	if err != nil {
		t.Fatal(err)
	}

	evaluations, completed, rejected := 0, 0, 0
	for _, testCase := range manifest.Cases {
		for index, evaluation := range testCase.Evaluations {
			evaluations++
			name := testCase.ID
			if len(testCase.Evaluations) > 1 {
				name += "/" + evaluation.ProfileID + "/" + string(rune('a'+index))
			}
			t.Run(name, func(t *testing.T) {
				if testCase.Kind == "canonicalization" {
					completed++
					assertCanonicalizationEvaluation(t, evaluation)
					return
				}
				kind := signedDocumentKind(t, evaluation.ProfileID)
				raw, err := missionweaveprotocol.ReadProtocolFile(evaluation.Document)
				if err != nil {
					t.Fatal(err)
				}
				registry, err := missionweaveprotocol.ReadProtocolFile(evaluation.Registry)
				if err != nil {
					t.Fatal(err)
				}
				verified, err := codec.Verify(kind, raw, fixtureKeyResolver{registry: registry})
				if evaluation.Expect.Stage != "complete" {
					rejected++
					assertRejectedEvaluation(t, err, evaluation.Expect)
					return
				}
				completed++
				if err != nil {
					t.Fatalf("expected complete verification: %v", err)
				}
				assertVerifiedEvaluation(t, verified, evaluation)
				assertSigningEvaluation(t, codec, kind, evaluation)
			})
		}
	}
	if evaluations != 58 || completed != 12 || rejected != 46 {
		t.Fatalf(
			"cryptography counts = %d evaluations, %d complete, %d rejected; want 58/12/46",
			evaluations,
			completed,
			rejected,
		)
	}
}

func assertCanonicalizationEvaluation(t *testing.T, evaluation cryptographyEvaluation) {
	t.Helper()
	input, err := missionweaveprotocol.ReadProtocolFile(evaluation.Input)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := missionweaveprotocol.ReadProtocolFile(evaluation.ExpectedJCS)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := missionweaveprotocol.CanonicalizeJSON(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != string(expected) {
		t.Fatal("RFC 8785 canonical bytes differ from the protocol artifact")
	}
	hash, err := missionweaveprotocol.CanonicalHash(input)
	if err != nil {
		t.Fatal(err)
	}
	if hash != evaluation.SHA256 {
		t.Fatalf("canonical hash = %s; want %s", hash, evaluation.SHA256)
	}
}

func assertRejectedEvaluation(t *testing.T, err error, expected cryptographyExpectation) {
	t.Helper()
	var failure *missionweaveprotocol.SignedDocumentVerificationError
	if !errors.As(err, &failure) {
		t.Fatalf("expected SignedDocumentVerificationError, got %T: %v", err, err)
	}
	if failure.ProtectedDiagnostic().Stage() != missionweaveprotocol.VerificationStage(expected.Stage) {
		t.Fatalf(
			"first failure stage = %s (%s); want %s",
			failure.ProtectedDiagnostic().Stage(),
			failure.ProtectedDiagnostic().Reason(),
			expected.Stage,
		)
	}
	if expected.WireCode == nil || string(failure.WireCode()) != *expected.WireCode {
		t.Fatalf("wire code = %s; want %v", failure.WireCode(), expected.WireCode)
	}
}

func assertVerifiedEvaluation(
	t *testing.T,
	verified *missionweaveprotocol.VerifiedSignedDocument,
	evaluation cryptographyEvaluation,
) {
	t.Helper()
	expected := evaluation.Expect.Verified
	if expected == nil {
		t.Fatal("complete evaluation has no verified evidence")
	}
	expectedSigningBytes, err := missionweaveprotocol.ReadProtocolFile(expected.SigningBytes)
	if err != nil {
		t.Fatal(err)
	}
	if string(verified.SigningBytes()) != string(expectedSigningBytes) {
		t.Fatal("signing bytes differ from the digest-protected artifact")
	}
	if verified.SigningHash() != expected.SigningHash ||
		verified.CompleteHash() != expected.SignedDocumentHash ||
		verified.ProtectedTime() != expected.ProtectedTime {
		t.Fatalf(
			"verified scalar evidence mismatch: signing=%s complete=%s protected=%s",
			verified.SigningHash(),
			verified.CompleteHash(),
			verified.ProtectedTime(),
		)
	}
	signature := verified.Signature()
	if signature.KeyID() != expected.KeyID || signature.Value() != expected.Signature {
		t.Fatal("signature evidence differs from the manifest")
	}
	key := verified.ResolvedKey()
	if key.KeyID() != expected.KeyID || key.Principal() != (missionweaveprotocol.Principal{
		Type: expected.Principal.Type,
		ID:   expected.Principal.ID,
	}) {
		t.Fatal("resolved key or Principal evidence differs from the manifest")
	}
}

func assertSigningEvaluation(
	t *testing.T,
	codec *missionweaveprotocol.SignedDocumentCodec,
	kind missionweaveprotocol.SignedDocumentKind,
	evaluation cryptographyEvaluation,
) {
	t.Helper()
	expected := readJSONObject(t, evaluation.Document)
	unsigned := readJSONObject(t, evaluation.Document)
	delete(unsigned, "signature")
	fixture := readJSONObject(t, evaluation.SigningKey)
	seed, err := base64.RawURLEncoding.Strict().DecodeString(fixture["seed"].(string))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := codec.Sign(kind, unsigned, fixtureSigningKey{
		keyID:      fixture["keyId"].(string),
		privateKey: ed25519.NewKeyFromSeed(seed),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatal("Sign did not reproduce the protocol-owned signed document")
	}
}

func signedDocumentKind(t *testing.T, profileID string) missionweaveprotocol.SignedDocumentKind {
	t.Helper()
	kinds := map[string]missionweaveprotocol.SignedDocumentKind{
		"agent-card":        missionweaveprotocol.SignedDocumentAgentCard,
		"approval":          missionweaveprotocol.SignedDocumentApproval,
		"artifact":          missionweaveprotocol.SignedDocumentArtifact,
		"command":           missionweaveprotocol.SignedDocumentCommand,
		"context-package":   missionweaveprotocol.SignedDocumentContextPackage,
		"event":             missionweaveprotocol.SignedDocumentEvent,
		"evidence":          missionweaveprotocol.SignedDocumentEvidence,
		"extension-profile": missionweaveprotocol.SignedDocumentExtensionProfile,
		"group-snapshot":    missionweaveprotocol.SignedDocumentGroupSnapshot,
	}
	kind, ok := kinds[profileID]
	if !ok {
		t.Fatalf("unknown signed-document profile %q", profileID)
	}
	return kind
}
