package missionweaveprotocol_test

import (
	"io/fs"
	"testing"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func TestEmbeddedProtocolBundleMatchesPin(t *testing.T) {
	if err := missionweaveprotocol.VerifyProtocolBundle(); err != nil {
		t.Fatal(err)
	}
	pin, err := missionweaveprotocol.CurrentProtocolPin()
	if err != nil {
		t.Fatal(err)
	}
	if pin.ProtocolVersion != "0.1" {
		t.Fatalf("unexpected protocol version %q", pin.ProtocolVersion)
	}
	if pin.WireNamespace != "missionweaveprotocol" {
		t.Fatalf("unexpected wire namespace %q", pin.WireNamespace)
	}

	var JSONFiles int
	err = fs.WalkDir(missionweaveprotocol.ProtocolFS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		isLegacyProtocolArtifact := name == "PROTOCOL_PIN.json" ||
			len(name) > len("schemas/") && name[:len("schemas/")] == "schemas/" ||
			len(name) > len("conformance/") && name[:len("conformance/")] == "conformance/"
		if isLegacyProtocolArtifact && !entry.IsDir() && len(entry.Name()) > 5 && entry.Name()[len(entry.Name())-5:] == ".json" {
			JSONFiles++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if JSONFiles != 79 { // 78 protocol artifacts plus PROTOCOL_PIN.json.
		t.Fatalf("expected 79 embedded JSON files, got %d", JSONFiles)
	}
}

func TestEmbeddedCryptographyBundleMatchesPin(t *testing.T) {
	if err := missionweaveprotocol.VerifyCryptographyBundle(); err != nil {
		t.Fatal(err)
	}
	pin, err := missionweaveprotocol.CurrentProtocolPin()
	if err != nil {
		t.Fatal(err)
	}
	cryptography := pin.Cryptography
	if cryptography.Path != "cryptography/manifest.json" {
		t.Fatalf("unexpected cryptography manifest path %q", cryptography.Path)
	}
	if cryptography.SourceCommit != "235aee85ba88934641822e1639e08efd2c9e29b6" {
		t.Fatalf("unexpected cryptography source commit %q", cryptography.SourceCommit)
	}
	if cryptography.ProfileID != "missionweaveprotocol.signed-document-verification.v0.1" {
		t.Fatalf("unexpected cryptography profile ID %q", cryptography.ProfileID)
	}
	if cryptography.ManifestVersion != 1 || cryptography.ArtifactCount != 94 || cryptography.CaseCount != 22 || cryptography.EvaluationCount != 58 {
		t.Fatalf("unexpected cryptography counts: %+v", cryptography)
	}
	if cryptography.ArtifactDigest != "sha256:159a4900987723537d0d110ec6724c5e1ee52854951a9c69278386d751baae08" {
		t.Fatalf("unexpected cryptography artifact digest %q", cryptography.ArtifactDigest)
	}
	for _, name := range []string{
		"cryptography/vectors/signed-documents/invalid/command-invalid-utf8.bin",
		"cryptography/vectors/canonicalization/command.signing.jcs",
		"cryptography/README.md",
	} {
		if _, err := missionweaveprotocol.ReadProtocolFile(name); err != nil {
			t.Fatalf("embedded cryptography resource %q is unavailable: %v", name, err)
		}
	}
}

func TestReadProtocolFileRejectsTraversal(t *testing.T) {
	if _, err := missionweaveprotocol.ReadProtocolFile("../PROTOCOL_PIN.json"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, err := missionweaveprotocol.ReadProtocolFile("README.md"); err == nil {
		t.Fatal("expected non-protocol path to be rejected")
	}
}
