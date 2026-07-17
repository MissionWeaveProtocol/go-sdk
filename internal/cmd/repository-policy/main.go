package main

import (
	"fmt"
	"os"

	"github.com/missionweaveprotocol/go-sdk/internal/repositorypolicy"
)

func main() {
	violations, err := repositorypolicy.Check(os.DirFS("."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "repository policy failed: %v\n", err)
		os.Exit(1)
	}
	if len(violations) != 0 {
		fmt.Fprintln(os.Stderr, "Repository vocabulary policy violations:")
		for _, violation := range violations {
			fmt.Fprintf(os.Stderr, "  %s\n", violation.Error())
		}
		os.Exit(1)
	}
	fmt.Println("Repository vocabulary policy passed.")
}
