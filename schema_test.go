package missionweaveprotocol_test

import (
	"bytes"
	"testing"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func TestSchemaCatalogValidatesEmbeddedDocumentsAndFormats(t *testing.T) {
	catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
	if err != nil {
		t.Fatal(err)
	}
	valid, err := missionweaveprotocol.ReadProtocolFile("conformance/vectors/valid/command.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Validate("command.schema.json", valid); err != nil {
		t.Fatal(err)
	}
	invalidTime := bytes.Replace(valid, []byte("2026-07-15T00:00:00Z"), []byte("not-a-date-time"), 1)
	if bytes.Equal(invalidTime, valid) {
		t.Fatal("test fixture timestamp was not found")
	}
	if err := catalog.Validate("schemas/command.schema.json", invalidTime); err == nil {
		t.Fatal("format-invalid date-time was accepted")
	}
}

func TestSchemaCatalogRejectsUnknownSchema(t *testing.T) {
	catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Validate("unknown.schema.json", []byte(`{}`)); err == nil {
		t.Fatal("unknown schema was accepted")
	}
}
