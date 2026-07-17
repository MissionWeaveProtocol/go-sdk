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
	err = fs.WalkDir(missionweaveprotocol.ProtocolFS(), ".", func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && len(entry.Name()) > 5 && entry.Name()[len(entry.Name())-5:] == ".json" {
			JSONFiles++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if JSONFiles != 66 { // 65 protocol artifacts plus PROTOCOL_PIN.json.
		t.Fatalf("expected 66 embedded JSON files, got %d", JSONFiles)
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
