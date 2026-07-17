package missionweaveprotocol_test

import (
	"testing"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func TestCanonicalizeJSONMatchesRFC8785NumberRendering(t *testing.T) {
	document := []byte(`{"numbers":[333333333.33333329,1E30,4.50,2e-3,1e-27],"z":2,"a":true}`)
	want := `{"a":true,"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],"z":2}`

	canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
	if err != nil {
		t.Fatal(err)
	}
	if string(canonical) != want {
		t.Fatalf("canonical JSON mismatch\nwant: %s\n got: %s", want, canonical)
	}
}

func TestCanonicalHashIgnoresObjectMemberOrder(t *testing.T) {
	left, err := missionweaveprotocol.CanonicalHash([]byte(`{"z":2,"a":{"enabled":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	right, err := missionweaveprotocol.CanonicalHash([]byte(`{"a":{"enabled":true},"z":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("canonical hashes differ: %s != %s", left, right)
	}
}
