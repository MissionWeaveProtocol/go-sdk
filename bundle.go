package missionweaveprotocol

import (
	"bytes"
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

var expectedCryptographyPin = CryptographyPin{
	Path:            "cryptography/manifest.json",
	SourceCommit:    "235aee85ba88934641822e1639e08efd2c9e29b6",
	ProfileID:       "missionweaveprotocol.signed-document-verification.v0.1",
	ManifestVersion: 1,
	ArtifactDigest:  "sha256:487e18c1ea7053432953f28d1496ae4fdb8e9d42c2eeb8e94f9b21f8cc2596a2",
	ArtifactCount:   94,
	CaseCount:       22,
	EvaluationCount: 58,
}

//go:embed PROTOCOL_PIN.json schemas/*.json conformance/manifest.json conformance/vectors/valid/*.json conformance/vectors/invalid/*.json cryptography
var embeddedProtocolBundle embed.FS

// PinnedArtifact records one byte-exact protocol artifact tree.
type PinnedArtifact struct {
	Path   string `json:"path"`
	Files  int    `json:"files"`
	SHA256 string `json:"sha256"`
}

// CryptographyPin identifies the independent signed-document cryptography bundle.
type CryptographyPin struct {
	Path            string `json:"path"`
	SourceCommit    string `json:"sourceCommit"`
	ProfileID       string `json:"profileId"`
	ManifestVersion int    `json:"manifestVersion"`
	ArtifactDigest  string `json:"artifactDigest"`
	ArtifactCount   int    `json:"artifactCount"`
	CaseCount       int    `json:"caseCount"`
	EvaluationCount int    `json:"evaluationCount"`
}

// ProtocolPin identifies the normative protocol source and its vendored artifact digests.
type ProtocolPin struct {
	Repository      string                    `json:"repository"`
	Commit          string                    `json:"commit"`
	ProtocolVersion string                    `json:"protocolVersion"`
	WireNamespace   string                    `json:"wireNamespace"`
	Artifacts       map[string]PinnedArtifact `json:"artifacts"`
	Cryptography    CryptographyPin           `json:"cryptography"`
	BundleSHA256    string                    `json:"bundleSha256"`
}

type cryptographyManifest struct {
	ManifestVersion int                    `json:"manifestVersion"`
	ProfileID       string                 `json:"profileId"`
	ProtocolVersion string                 `json:"protocolVersion"`
	ArtifactDigest  string                 `json:"artifactDigest"`
	Artifacts       []cryptographyArtifact `json:"artifacts"`
	Cases           []cryptographyCase     `json:"cases"`
}

type cryptographyArtifact struct {
	Path       string `json:"path"`
	ByteLength int    `json:"byteLength"`
	SHA256     string `json:"sha256"`
}

type cryptographyCase struct {
	Evaluations []json.RawMessage `json:"evaluations"`
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
	pin, err := loadProtocolPin()
	if err != nil {
		return ProtocolPin{}, err
	}
	if err := validatePin(pin); err != nil {
		return ProtocolPin{}, err
	}
	return pin, nil
}

func loadProtocolPin() (ProtocolPin, error) {
	contents, err := ReadProtocolFile(protocolPinPath)
	if err != nil {
		return ProtocolPin{}, err
	}
	var pin ProtocolPin
	if err := json.Unmarshal(contents, &pin); err != nil {
		return ProtocolPin{}, fmt.Errorf("decode protocol pin: %w", err)
	}
	return pin, nil
}

func loadStrictProtocolPin() (ProtocolPin, error) {
	contents, err := ReadProtocolFile(protocolPinPath)
	if err != nil {
		return ProtocolPin{}, err
	}
	if _, err := DecodeJSON(contents); err != nil {
		return ProtocolPin{}, fmt.Errorf("decode protocol pin strictly: %w", err)
	}
	var pin ProtocolPin
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pin); err != nil {
		return ProtocolPin{}, fmt.Errorf("decode protocol pin: %w", err)
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

// VerifyCryptographyBundle verifies the independent signed-document cryptography manifest and
// every digest-protected artifact embedded with this SDK build.
func VerifyCryptographyBundle() error {
	pin, err := loadStrictProtocolPin()
	if err != nil {
		return err
	}
	if err := validateCryptographyPin(pin.Cryptography); err != nil {
		return err
	}
	manifestBytes, err := ReadProtocolFile(pin.Cryptography.Path)
	if err != nil {
		return err
	}
	parsed, err := DecodeJSON(manifestBytes)
	if err != nil {
		return fmt.Errorf("decode cryptography manifest: %w", err)
	}
	manifestObject, ok := parsed.(map[string]any)
	if !ok {
		return errors.New("cryptography manifest must be a JSON object")
	}
	var manifest cryptographyManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("decode cryptography manifest fields: %w", err)
	}
	if err := verifyCryptographyManifestMetadata(pin, manifest); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		if err := validateCryptographyArtifactPath(artifact.Path); err != nil {
			return err
		}
		if _, duplicate := seen[artifact.Path]; duplicate {
			return fmt.Errorf("cryptography manifest repeats artifact path %q", artifact.Path)
		}
		seen[artifact.Path] = struct{}{}
		contents, err := ReadProtocolFile(artifact.Path)
		if err != nil {
			return fmt.Errorf("verify cryptography artifact %q: %w", artifact.Path, err)
		}
		if len(contents) != artifact.ByteLength {
			return fmt.Errorf(
				"cryptography artifact %q byte length mismatch: manifest=%d embedded=%d",
				artifact.Path,
				artifact.ByteLength,
				len(contents),
			)
		}
		actual := sha256Identifier(contents)
		if actual != artifact.SHA256 {
			return fmt.Errorf(
				"cryptography artifact %q digest mismatch: manifest=%s embedded=%s",
				artifact.Path,
				artifact.SHA256,
				actual,
			)
		}
	}

	delete(manifestObject, "artifactDigest")
	canonical, err := MarshalCanonicalJSON(manifestObject)
	if err != nil {
		return fmt.Errorf("canonicalize cryptography manifest: %w", err)
	}
	actualDigest := sha256Identifier(canonical)
	if actualDigest != pin.Cryptography.ArtifactDigest {
		return fmt.Errorf(
			"cryptography manifest digest mismatch: pin=%s embedded=%s",
			pin.Cryptography.ArtifactDigest,
			actualDigest,
		)
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

func validateCryptographyPin(pin CryptographyPin) error {
	if pin != expectedCryptographyPin {
		return fmt.Errorf("protocol pin cryptography entry does not match the published bundle: %+v", pin)
	}
	return nil
}

func verifyCryptographyManifestMetadata(pin ProtocolPin, manifest cryptographyManifest) error {
	cryptography := pin.Cryptography
	if manifest.ManifestVersion != cryptography.ManifestVersion ||
		manifest.ProfileID != cryptography.ProfileID ||
		manifest.ProtocolVersion != pin.ProtocolVersion ||
		manifest.ArtifactDigest != cryptography.ArtifactDigest {
		return errors.New("cryptography manifest identity does not match PROTOCOL_PIN.json")
	}
	if len(manifest.Artifacts) != cryptography.ArtifactCount {
		return fmt.Errorf(
			"cryptography artifact count mismatch: pin=%d manifest=%d",
			cryptography.ArtifactCount,
			len(manifest.Artifacts),
		)
	}
	if len(manifest.Cases) != cryptography.CaseCount {
		return fmt.Errorf(
			"cryptography case count mismatch: pin=%d manifest=%d",
			cryptography.CaseCount,
			len(manifest.Cases),
		)
	}
	evaluations := 0
	for _, testCase := range manifest.Cases {
		evaluations += len(testCase.Evaluations)
	}
	if evaluations != cryptography.EvaluationCount {
		return fmt.Errorf(
			"cryptography evaluation count mismatch: pin=%d manifest=%d",
			cryptography.EvaluationCount,
			evaluations,
		)
	}
	return nil
}

func validateCryptographyArtifactPath(name string) error {
	if !fs.ValidPath(name) || path.Clean(name) != name || strings.Contains(name, "\\") {
		return fmt.Errorf("cryptography manifest contains unsafe artifact path %q", name)
	}
	if name == "cryptography/README.md" || name == "cryptography/manifest.json" {
		return fmt.Errorf("cryptography manifest contains non-artifact path %q", name)
	}
	if !strings.HasPrefix(name, "cryptography/") && !strings.HasPrefix(name, "schemas/") {
		return fmt.Errorf("cryptography manifest artifact path is outside pinned roots: %q", name)
	}
	return nil
}

func sha256Identifier(contents []byte) string {
	digest := sha256.Sum256(contents)
	return "sha256:" + hex.EncodeToString(digest[:])
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
	return name == protocolPinPath ||
		strings.HasPrefix(name, "schemas/") ||
		strings.HasPrefix(name, "conformance/") ||
		strings.HasPrefix(name, "cryptography/")
}
