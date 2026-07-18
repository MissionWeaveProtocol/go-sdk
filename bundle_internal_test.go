package missionweaveprotocol

import "testing"

func TestCryptographyPinRejectsPublishedIdentityDrift(t *testing.T) {
	t.Parallel()

	mutations := []CryptographyPin{
		func() CryptographyPin {
			pin := expectedCryptographyPin
			pin.SourceCommit = "335aee85ba88934641822e1639e08efd2c9e29b6"
			return pin
		}(),
		func() CryptographyPin {
			pin := expectedCryptographyPin
			pin.ArtifactDigest = "sha256:587e18c1ea7053432953f28d1496ae4fdb8e9d42c2eeb8e94f9b21f8cc2596a2"
			return pin
		}(),
		func() CryptographyPin {
			pin := expectedCryptographyPin
			pin.EvaluationCount++
			return pin
		}(),
	}

	for _, pin := range mutations {
		if err := validateCryptographyPin(pin); err == nil {
			t.Fatalf("expected cryptography pin drift to be rejected: %+v", pin)
		}
	}
}

func TestCryptographyArtifactPathsStayWithinPinnedRoots(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"../schemas/command.schema.json",
		"/schemas/command.schema.json",
		"cryptography\\..\\PROTOCOL_PIN.json",
		"conformance/manifest.json",
		"cryptography/README.md",
		"cryptography/manifest.json",
	} {
		if err := validateCryptographyArtifactPath(name); err == nil {
			t.Errorf("expected artifact path %q to be rejected", name)
		}
	}
	for _, name := range []string{
		"schemas/command.schema.json",
		"cryptography/keys/registry-valid.json",
		"cryptography/vectors/signed-documents/invalid/command-invalid-utf8.bin",
		"cryptography/vectors/canonicalization/command.signing.jcs",
	} {
		if err := validateCryptographyArtifactPath(name); err != nil {
			t.Errorf("expected artifact path %q to be accepted: %v", name, err)
		}
	}
}
