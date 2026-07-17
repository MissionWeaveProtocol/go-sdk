// Package repositorypolicy enforces canonical repository naming and documentation vocabulary.
package repositorypolicy

import (
	"bytes"
	"fmt"
	"io/fs"
	"regexp"
	"strings"
)

var (
	legacyShort  = strings.Join([]string{"aw", "gp"}, "")
	legacyPhrase = regexp.MustCompile(
		`(?i)\b` + strings.Join([]string{"agent", "workgroup", "protocol"}, `[\s_-]+`) + `\b`,
	)
	decisionWord = strings.Join([]string{"a", "dr"}, "")
	decisionTerm = regexp.MustCompile(
		`(?i)\b` + strings.Join([]string{"architecture", "decision", "record"}, `[\s_-]+`) + `s?\b`,
	)
	decisionReference   = regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(decisionWord) + `s?\b`)
	displayFragment     = strings.Join([]string{"Mission", "Weave"}, "")
	machineFragment     = strings.Join([]string{"mission", "weave"}, "")
	environmentFragment = strings.Join([]string{"MISSION", "WEAVE"}, "")
)

var ignoredDirectories = map[string]struct{}{
	".git":   {},
	"bin":    {},
	"dist":   {},
	"vendor": {},
}

// Violation describes one canonical vocabulary failure.
type Violation struct {
	Path string
	Kind string
}

func (v Violation) Error() string {
	return fmt.Sprintf("%s: %s", v.Path, v.Kind)
}

// Check scans a repository filesystem and returns every canonical vocabulary violation.
func Check(repository fs.FS) ([]Violation, error) {
	var violations []Violation
	err := fs.WalkDir(repository, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if _, ignored := ignoredDirectories[entry.Name()]; ignored && path != "." {
				return fs.SkipDir
			}
			if strings.EqualFold(entry.Name(), decisionWord) {
				violations = append(violations, Violation{Path: path, Kind: "retired documentation path"})
			}
			return nil
		}

		inspect(path, path, &violations)
		contents, err := fs.ReadFile(repository, path)
		if err != nil {
			return err
		}
		if bytes.IndexByte(contents, 0) >= 0 {
			return nil
		}
		inspect(path, string(contents), &violations)
		return nil
	})
	return violations, err
}

func inspect(path, value string, violations *[]Violation) {
	lower := strings.ToLower(value)
	if strings.Contains(lower, legacyShort) {
		*violations = append(*violations, Violation{Path: path, Kind: "retired identifier"})
	}
	if legacyPhrase.MatchString(value) {
		*violations = append(*violations, Violation{Path: path, Kind: "retired protocol name"})
	}
	if containsIncomplete(value, displayFragment, displayFragment+"Protocol") ||
		containsIncomplete(value, machineFragment, machineFragment+"protocol") ||
		containsIncomplete(value, environmentFragment, environmentFragment+"PROTOCOL") {
		*violations = append(*violations, Violation{Path: path, Kind: "incomplete product identity"})
	}
	if decisionReference.MatchString(value) || decisionTerm.MatchString(value) {
		*violations = append(*violations, Violation{Path: path, Kind: "retired documentation reference"})
	}
}

func containsIncomplete(value, fragment, complete string) bool {
	for offset := 0; ; {
		index := strings.Index(value[offset:], fragment)
		if index < 0 {
			return false
		}
		index += offset
		if !strings.HasPrefix(value[index:], complete) {
			return true
		}
		offset = index + len(fragment)
	}
}
