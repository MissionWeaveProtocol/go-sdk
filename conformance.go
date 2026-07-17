package missionweaveprotocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
)

const conformanceManifestPath = "conformance/manifest.json"

// VectorResult records the expected and observed validity of one conformance vector.
type VectorResult struct {
	Name          string
	ExpectedValid bool
	ActualValid   bool
	Error         string
}

// Passed reports whether the observed validity matched the manifest.
func (result VectorResult) Passed() bool {
	return result.ExpectedValid == result.ActualValid
}

// ConformanceReport records every manifest result.
type ConformanceReport struct {
	Results []VectorResult
}

// Passed reports whether every conformance vector matched its expected validity.
func (report ConformanceReport) Passed() bool {
	for _, result := range report.Results {
		if !result.Passed() {
			return false
		}
	}
	return true
}

// Summary returns the stable human-readable conformance count.
func (report ConformanceReport) Summary() string {
	passed := 0
	for _, result := range report.Results {
		if result.Passed() {
			passed++
		}
	}
	return fmt.Sprintf("%d/%d conformance vectors passed", passed, len(report.Results))
}

type manifestVector struct {
	Name     string `json:"name"`
	Schema   string `json:"schema"`
	Instance string `json:"instance"`
	Valid    bool   `json:"valid"`
}

// RunConformance validates the complete manifest found in source using an offline SchemaCatalog.
func RunConformance(source fs.FS) (ConformanceReport, error) {
	if source == nil {
		return ConformanceReport{}, errors.New("conformance source must not be nil")
	}
	manifestDocument, err := fs.ReadFile(source, conformanceManifestPath)
	if err != nil {
		return ConformanceReport{}, fmt.Errorf("read conformance manifest: %w", err)
	}
	if _, err := DecodeJSON(manifestDocument); err != nil {
		return ConformanceReport{}, fmt.Errorf("parse conformance manifest: %w", err)
	}
	var vectors []manifestVector
	if err := json.Unmarshal(manifestDocument, &vectors); err != nil {
		return ConformanceReport{}, fmt.Errorf("decode conformance manifest: %w", err)
	}
	if len(vectors) == 0 {
		return ConformanceReport{}, errors.New("conformance manifest contains no vectors")
	}
	catalog, err := NewSchemaCatalog(source)
	if err != nil {
		return ConformanceReport{}, err
	}

	report := ConformanceReport{Results: make([]VectorResult, 0, len(vectors))}
	for index, vector := range vectors {
		if vector.Name == "" || vector.Schema == "" || vector.Instance == "" {
			return ConformanceReport{}, fmt.Errorf("conformance manifest entry %d is incomplete", index)
		}
		instance, err := fs.ReadFile(source, vector.Instance)
		if err != nil {
			return ConformanceReport{}, fmt.Errorf("read conformance vector %s: %w", vector.Name, err)
		}
		validationErr := catalog.Validate(vector.Schema, instance)
		actualValid := validationErr == nil
		detail := ""
		if validationErr != nil {
			var documentError *DocumentValidationError
			if !errors.As(validationErr, &documentError) {
				return ConformanceReport{}, fmt.Errorf("run conformance vector %s: %w", vector.Name, validationErr)
			}
			detail = validationErr.Error()
		}
		report.Results = append(report.Results, VectorResult{
			Name:          vector.Name,
			ExpectedValid: vector.Valid,
			ActualValid:   actualValid,
			Error:         detail,
		})
	}
	return report, nil
}

// RunEmbeddedConformance validates the complete protocol bundle embedded in this SDK build.
func RunEmbeddedConformance() (ConformanceReport, error) {
	return RunConformance(ProtocolFS())
}
