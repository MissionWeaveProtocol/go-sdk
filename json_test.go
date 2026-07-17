package missionweaveprotocol_test

import (
	"strings"
	"testing"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func TestDecodeJSONRejectsDuplicateMembersAtEveryDepth(t *testing.T) {
	for _, document := range []string{
		`{"value":1,"value":2}`,
		`{"outer":{"value":1,"value":2}}`,
		`{"\u0076alue":1,"value":2}`,
	} {
		if _, err := missionweaveprotocol.DecodeJSON([]byte(document)); err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("expected duplicate-member error for %s, got %v", document, err)
		}
	}
}

func TestDecodeJSONRejectsInvalidUTF8AndTrailingValues(t *testing.T) {
	invalidUTF8 := []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}
	if _, err := missionweaveprotocol.DecodeJSON(invalidUTF8); err == nil {
		t.Fatal("expected invalid UTF-8 to be rejected")
	}
	if _, err := missionweaveprotocol.DecodeJSON([]byte(`{} {}`)); err == nil {
		t.Fatal("expected multiple top-level values to be rejected")
	}
}
