package main

import (
	"fmt"
	"os"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func main() {
	if err := missionweaveprotocol.VerifyProtocolBundle(); err != nil {
		fmt.Fprintf(os.Stderr, "embedded protocol bundle verification failed: %v\n", err)
		os.Exit(1)
	}
	if err := missionweaveprotocol.VerifyCryptographyBundle(); err != nil {
		fmt.Fprintf(os.Stderr, "embedded cryptography bundle verification failed: %v\n", err)
		os.Exit(1)
	}
	pin, err := missionweaveprotocol.CurrentProtocolPin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "embedded protocol pin failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("MissionWeaveProtocol %s bundle verified at %s\n", pin.ProtocolVersion, pin.Commit)
}
