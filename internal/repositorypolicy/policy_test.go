package repositorypolicy

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestCheckAcceptsCanonicalIdentity(t *testing.T) {
	repository := fstest.MapFS{
		"README.md": {Data: []byte("MissionWeaveProtocol missionweaveprotocol MISSIONWEAVEPROTOCOL")},
	}

	violations, err := Check(repository)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %v", violations)
	}
}

func TestCheckRejectsRetiredAndIncompleteIdentity(t *testing.T) {
	retired := strings.Join([]string{"Agent", "Workgroup", "Protocol"}, " ")
	incomplete := strings.Join([]string{"Mission", "Weave"}, "")
	repository := fstest.MapFS{
		"README.md": {Data: []byte(retired + "\n" + incomplete)},
	}

	violations, err := Check(repository)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) < 2 {
		t.Fatalf("expected retired and incomplete identity failures, got %v", violations)
	}
}
