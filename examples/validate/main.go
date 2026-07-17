package main

import (
	"fmt"
	"log"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func main() {
	if err := missionweaveprotocol.VerifyProtocolBundle(); err != nil {
		log.Fatal(err)
	}
	pin, err := missionweaveprotocol.CurrentProtocolPin()
	if err != nil {
		log.Fatal(err)
	}

	catalog, err := missionweaveprotocol.NewEmbeddedSchemaCatalog()
	if err != nil {
		log.Fatal(err)
	}
	command, err := missionweaveprotocol.ReadProtocolFile("conformance/vectors/valid/command.json")
	if err != nil {
		log.Fatal(err)
	}
	if err := catalog.Validate("command.schema.json", command); err != nil {
		log.Fatal(err)
	}
	hash, err := missionweaveprotocol.CanonicalHash(command)
	if err != nil {
		log.Fatal(err)
	}

	codec, err := missionweaveprotocol.NewFrameCodec()
	if err != nil {
		log.Fatal(err)
	}
	frameDocument, err := missionweaveprotocol.ReadProtocolFile("conformance/vectors/valid/websocket-frame.json")
	if err != nil {
		log.Fatal(err)
	}
	frame, err := codec.DecodeFrame(frameDocument)
	if err != nil {
		log.Fatal(err)
	}
	encoded, err := codec.EncodeFrame(frame)
	if err != nil {
		log.Fatal(err)
	}

	report, err := missionweaveprotocol.RunEmbeddedConformance()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(
		"MissionWeaveProtocol %s: %s; command=%s; frame=%s\n",
		pin.ProtocolVersion,
		report.Summary(),
		hash,
		encoded,
	)
}
