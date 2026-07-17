package missionweaveprotocol_test

import (
	"reflect"
	"testing"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func TestFrameCodecRoundTripsSchemaValidCanonicalJSON(t *testing.T) {
	codec, err := missionweaveprotocol.NewFrameCodec()
	if err != nil {
		t.Fatal(err)
	}
	document, err := missionweaveprotocol.ReadProtocolFile("conformance/vectors/valid/websocket-frame.json")
	if err != nil {
		t.Fatal(err)
	}
	frame, err := codec.DecodeFrame(document)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := codec.EncodeFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := codec.DecodeFrame(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(frame, roundTrip) {
		t.Fatalf("frame round trip changed value\nwant: %#v\n got: %#v", frame, roundTrip)
	}
	canonical, err := missionweaveprotocol.CanonicalizeJSON(document)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != string(canonical) {
		t.Fatalf("frame was not canonically encoded\nwant: %s\n got: %s", canonical, encoded)
	}
}

func TestFrameCodecRejectsDuplicateUnknownAndInvalidUTF8Frames(t *testing.T) {
	codec, err := missionweaveprotocol.NewFrameCodec()
	if err != nil {
		t.Fatal(err)
	}
	duplicates := []byte(`{"protocolVersion":"0.1","protocolVersion":"0.1","frameId":"urn:x:1","frameType":"UNKNOWN"}`)
	if _, err := codec.DecodeFrame(duplicates); err == nil {
		t.Fatal("duplicate frame member was accepted")
	}
	unknown := []byte(`{"protocolVersion":"0.1","frameId":"urn:x:1","frameType":"UNKNOWN"}`)
	if _, err := codec.DecodeFrame(unknown); err == nil {
		t.Fatal("unknown frame type was accepted")
	}
	invalidUTF8 := append([]byte(`{"protocolVersion":"0.1","frameId":"urn:x:1","frameType":"`), 0xff)
	if _, err := codec.DecodeFrame(invalidUTF8); err == nil {
		t.Fatal("invalid UTF-8 frame was accepted")
	}
}

func TestFrameCodecPreservesExtensionDataWithoutPromotingCoreFields(t *testing.T) {
	codec, err := missionweaveprotocol.NewFrameCodec()
	if err != nil {
		t.Fatal(err)
	}
	commandDocument, err := missionweaveprotocol.ReadProtocolFile("conformance/vectors/valid/command.json")
	if err != nil {
		t.Fatal(err)
	}
	value, err := missionweaveprotocol.DecodeJSON(commandDocument)
	if err != nil {
		t.Fatal(err)
	}
	command := value.(map[string]any)
	extensions := map[string]any{
		"https://profiles.example/audit": map[string]any{
			"version":  "1.2.3",
			"critical": false,
			"data": map[string]any{
				"kind":    "mission.approved",
				"groupId": "urn:missionweaveprotocol:group:forged",
				"payload": map[string]any{"forged": true},
			},
		},
	}
	command["extensions"] = extensions
	frame := map[string]any{
		"protocolVersion": "0.1",
		"frameId":         "urn:uuid:00000000-0000-4000-8000-000000000010",
		"frameType":       "COMMAND",
		"command":         command,
	}

	encoded, err := codec.EncodeFrame(frame)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := codec.DecodeFrame(encoded)
	if err != nil {
		t.Fatal(err)
	}
	decodedCommand := decoded["command"].(map[string]any)
	if !reflect.DeepEqual(decodedCommand["extensions"], extensions) {
		t.Fatalf("extension data changed: %#v", decodedCommand["extensions"])
	}
	if decodedCommand["kind"] != command["kind"] || decodedCommand["groupId"] != command["groupId"] {
		t.Fatal("extension data promoted or replaced core command fields")
	}
}
