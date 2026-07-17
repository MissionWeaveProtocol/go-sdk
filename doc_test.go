package missionweaveprotocol_test

import (
	"testing"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func TestPublicPackageIdentity(t *testing.T) {
	if missionweaveprotocol.SDKVersion != "0.1.0" {
		t.Fatalf("unexpected SDK version %q", missionweaveprotocol.SDKVersion)
	}
}
