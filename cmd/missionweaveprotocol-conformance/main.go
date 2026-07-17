package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"

	missionweaveprotocol "github.com/missionweaveprotocol/go-sdk"
)

func main() {
	root := flag.String("root", "", "protocol repository or release-bundle root; defaults to embedded artifacts")
	flag.Parse()

	var source fs.FS = missionweaveprotocol.ProtocolFS()
	if *root != "" {
		source = os.DirFS(*root)
	}
	report, err := missionweaveprotocol.RunConformance(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "conformance failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(report.Summary())
	if report.Passed() {
		return
	}
	for _, result := range report.Results {
		if result.Passed() {
			continue
		}
		fmt.Fprintf(
			os.Stderr,
			"FAIL %s: expected valid=%v actual valid=%v: %s\n",
			result.Name,
			result.ExpectedValid,
			result.ActualValid,
			result.Error,
		)
	}
	os.Exit(1)
}
