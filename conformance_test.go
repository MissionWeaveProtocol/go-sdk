package missionweaveprotocol_test

import (
	"testing"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func TestEmbeddedConformanceManifest(t *testing.T) {
	report, err := missionweaveprotocol.RunEmbeddedConformance()
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 56 {
		t.Fatalf("expected 56 conformance vectors, got %d", len(report.Results))
	}
	if !report.Passed() {
		for _, result := range report.Results {
			if !result.Passed() {
				t.Errorf("%s: expected valid=%v actual valid=%v: %s", result.Name, result.ExpectedValid, result.ActualValid, result.Error)
			}
		}
	}
	if report.Summary() != "56/56 conformance vectors passed" {
		t.Fatalf("unexpected report summary %q", report.Summary())
	}
}
