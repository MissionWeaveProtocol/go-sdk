package missionweaveprotocol

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

const protocolPinPath = "PROTOCOL_PIN.json"

//go:embed PROTOCOL_PIN.json schemas/*.json conformance/manifest.json conformance/vectors/valid/*.json conformance/vectors/invalid/*.json
var embeddedProtocolBundle embed.FS

// PinnedArtifact records one byte-exact protocol artifact tree.
type PinnedArtifact struct {
	Path   string `json:"path"`
	Files  int    `json:"files"`
	SHA256 string `json:"sha256"`
}

// ProtocolPin identifies the normative protocol source and its vendored artifact digests.
type ProtocolPin struct {
	Repository      string                    `json:"repository"`
	Commit          string                    `json:"commit"`
	ProtocolVersion string                    `json:"protocolVersion"`
	WireNamespace   string                    `json:"wireNamespace"`
	Artifacts       map[string]PinnedArtifact `json:"artifacts"`
	BundleSHA256    string                    `json:"bundleSha256"`
}

// ProtocolFS returns the immutable embedded protocol artifact filesystem.
func ProtocolFS() fs.FS {
	return embeddedProtocolBundle
}

// ReadProtocolFile reads one embedded protocol file by its repository-relative logical path.
func ReadProtocolFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) || path.Clean(name) != name || !isProtocolPath(name) {
		return nil, fmt.Errorf("invalid protocol resource path %q", name)
	}
	contents, err := fs.ReadFile(embeddedProtocolBundle, name)
	if err != nil {
		return nil, fmt.Errorf("read protocol resource %q: %w", name, err)
	}
	return contents, nil
}

// CurrentProtocolPin loads the metadata embedded with this SDK build.
func CurrentProtocolPin() (ProtocolPin, error) {
	contents, err := ReadProtocolFile(protocolPinPath)
	if err != nil {
		return ProtocolPin{}, err
	}
	var pin ProtocolPin
	if err := json.Unmarshal(contents, &pin); err != nil {
		return ProtocolPin{}, fmt.Errorf("decode protocol pin: %w", err)
	}
	if err := validatePin(pin); err != nil {
		return ProtocolPin{}, err
	}
	return pin, nil
}

// VerifyProtocolBundle verifies file counts and all three digests from PROTOCOL_PIN.json.
func VerifyProtocolBundle() error {
	pin, err := CurrentProtocolPin()
	if err != nil {
		return err
	}
	var allPaths []string
	for _, name := range []string{"schemas", "conformance"} {
		artifact := pin.Artifacts[name]
		paths, digest, err := digestJSONTree(artifact.Path)
		if err != nil {
			return fmt.Errorf("verify %s artifact: %w", name, err)
		}
		if len(paths) != artifact.Files {
			return fmt.Errorf(
				"%s artifact file count mismatch: pin=%d embedded=%d",
				name,
				artifact.Files,
				len(paths),
			)
		}
		if digest != artifact.SHA256 {
			return fmt.Errorf("%s artifact digest mismatch: pin=%s embedded=%s", name, artifact.SHA256, digest)
		}
		allPaths = append(allPaths, paths...)
	}
	sort.Strings(allPaths)
	bundleDigest, err := digestPaths(allPaths)
	if err != nil {
		return fmt.Errorf("verify protocol bundle: %w", err)
	}
	if bundleDigest != pin.BundleSHA256 {
		return fmt.Errorf("protocol bundle digest mismatch: pin=%s embedded=%s", pin.BundleSHA256, bundleDigest)
	}
	return nil
}

func validatePin(pin ProtocolPin) error {
	if pin.Repository == "" || pin.Commit == "" || pin.ProtocolVersion == "" || pin.WireNamespace == "" {
		return errors.New("protocol pin identity fields must not be empty")
	}
	if len(pin.Artifacts) != 2 {
		return fmt.Errorf("protocol pin must contain schemas and conformance artifacts")
	}
	for name, expectedPath := range map[string]string{
		"schemas":     "schemas",
		"conformance": "conformance",
	} {
		artifact, ok := pin.Artifacts[name]
		if !ok {
			return fmt.Errorf("protocol pin lacks %s artifact", name)
		}
		if artifact.Path != expectedPath || artifact.Files <= 0 || len(artifact.SHA256) != sha256.Size*2 {
			return fmt.Errorf("protocol pin contains invalid %s artifact metadata", name)
		}
	}
	if len(pin.BundleSHA256) != sha256.Size*2 {
		return errors.New("protocol pin contains an invalid bundle digest")
	}
	return nil
}

func digestJSONTree(root string) ([]string, string, error) {
	var paths []string
	err := fs.WalkDir(embeddedProtocolBundle, root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			paths = append(paths, name)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	sort.Strings(paths)
	digest, err := digestPaths(paths)
	return paths, digest, err
}

func digestPaths(paths []string) (string, error) {
	digest := sha256.New()
	for _, name := range paths {
		contents, err := ReadProtocolFile(name)
		if err != nil {
			return "", err
		}
		digest.Write([]byte(name))
		digest.Write([]byte{0})
		digest.Write(contents)
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func isProtocolPath(name string) bool {
	return name == protocolPinPath || strings.HasPrefix(name, "schemas/") || strings.HasPrefix(name, "conformance/")
}
