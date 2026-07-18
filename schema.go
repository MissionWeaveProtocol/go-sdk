package missionweaveprotocol

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// DocumentValidationError reports that a JSON document failed strict parsing or one normative
// schema.
type DocumentValidationError struct {
	Schema string
	Cause  error
}

func (e *DocumentValidationError) Error() string {
	return fmt.Sprintf("%s validation failed: %v", e.Schema, e.Cause)
}

func (e *DocumentValidationError) Unwrap() error {
	return e.Cause
}

// SchemaCatalog owns one fully resolved, offline set of normative schemas.
type SchemaCatalog struct {
	schemas map[string]*jsonschema.Schema
}

// NewSchemaCatalog compiles every schema from source, registers references by $id, enables Draft
// 2020-12 format assertions, and forbids network loading.
func NewSchemaCatalog(source fs.FS) (*SchemaCatalog, error) {
	if source == nil {
		return nil, errors.New("schema source must not be nil")
	}
	paths, err := fs.Glob(source, "schemas/*.json")
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}
	if len(paths) == 0 {
		return nil, errors.New("schema source contains no schemas")
	}
	sort.Strings(paths)

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.RegisterFormat(&jsonschema.Format{
		Name:     "date-time",
		Validate: validateProtocolDateTimeFormat,
	})
	compiler.AssertFormat()
	compiler.UseLoader(offlineSchemaLoader{})
	compiler.UseRegexpEngine(compileECMAScript)

	identifiers := make(map[string]string, len(paths))
	for _, schemaPath := range paths {
		document, err := fs.ReadFile(source, schemaPath)
		if err != nil {
			return nil, fmt.Errorf("read schema %s: %w", schemaPath, err)
		}
		value, err := DecodeJSON(document)
		if err != nil {
			return nil, fmt.Errorf("parse schema %s: %w", schemaPath, err)
		}
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("schema %s is not a JSON object", schemaPath)
		}
		identifier, ok := object["$id"].(string)
		if !ok || identifier == "" {
			return nil, fmt.Errorf("schema %s lacks $id", schemaPath)
		}
		name := path.Base(schemaPath)
		if _, duplicate := identifiers[name]; duplicate {
			return nil, fmt.Errorf("duplicate schema name %s", name)
		}
		if err := compiler.AddResource(identifier, value); err != nil {
			return nil, fmt.Errorf("register schema %s: %w", schemaPath, err)
		}
		identifiers[name] = identifier
	}

	compiled := make(map[string]*jsonschema.Schema, len(identifiers))
	for name, identifier := range identifiers {
		schema, err := compiler.Compile(identifier)
		if err != nil {
			return nil, fmt.Errorf("compile schema %s: %w", name, err)
		}
		compiled[name] = schema
	}
	return &SchemaCatalog{schemas: compiled}, nil
}

// NewEmbeddedSchemaCatalog compiles the protocol schemas embedded in this SDK build.
func NewEmbeddedSchemaCatalog() (*SchemaCatalog, error) {
	return NewSchemaCatalog(ProtocolFS())
}

// Validate parses a strict JSON document and validates it against a named normative schema.
func (catalog *SchemaCatalog) Validate(schemaName string, document []byte) error {
	value, err := DecodeJSON(document)
	if err != nil {
		return &DocumentValidationError{Schema: schemaName, Cause: err}
	}
	return catalog.validateValue(schemaName, value)
}

func (catalog *SchemaCatalog) validateValue(schemaName string, value any) error {
	name, err := normalizeSchemaName(schemaName)
	if err != nil {
		return err
	}
	schema, ok := catalog.schemas[name]
	if !ok {
		return fmt.Errorf("unknown schema %q", schemaName)
	}
	if err := schema.Validate(value); err != nil {
		return &DocumentValidationError{Schema: name, Cause: err}
	}
	return nil
}

func normalizeSchemaName(name string) (string, error) {
	name = strings.TrimPrefix(name, "schemas/")
	if name == "" || path.Base(name) != name || !strings.HasSuffix(name, ".schema.json") {
		return "", fmt.Errorf("invalid schema name %q", name)
	}
	return name, nil
}

type offlineSchemaLoader struct{}

func (offlineSchemaLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("offline schema catalog cannot load %s", url)
}

type ecmaRegexp regexp2.Regexp

func (expression *ecmaRegexp) MatchString(value string) bool {
	matched, err := (*regexp2.Regexp)(expression).MatchString(value)
	return err == nil && matched
}

func (expression *ecmaRegexp) String() string {
	return (*regexp2.Regexp)(expression).String()
}

func compileECMAScript(pattern string) (jsonschema.Regexp, error) {
	expression, err := regexp2.Compile(pattern, regexp2.ECMAScript)
	if err != nil {
		return nil, err
	}
	return (*ecmaRegexp)(expression), nil
}
