package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func main() {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	document := []byte(`{"protocolVersion":"0.1","value":"example","signature":{"value":"excluded"}}`)
	signature, err := missionweaveprotocol.SignDocument(privateKey, document)
	if err != nil {
		log.Fatal(err)
	}
	verified, err := missionweaveprotocol.VerifyDocument(publicKey, document, signature)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("signature=%s verified=%v\n", signature, verified)
}
