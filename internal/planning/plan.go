// Package planning describes generation intent without filesystem, network, or execution effects.
package planning

import "github.com/ProvisionLabs/provisionhub/internal/catalog"

type Status string

const (
	StatusPlanned Status = "planned"
	StatusInvalid Status = "invalid"
)

type OperationType string

const (
	GenerateBackend   OperationType = "generate_backend"
	GenerateFrontend  OperationType = "generate_frontend"
	IntegrateDatabase OperationType = "integrate_database"
	GenerateFile      OperationType = "generate_file"
)

type OperationStatus string

const (
	OperationUnavailable OperationStatus = "unavailable"
	OperationSupported   OperationStatus = "supported"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type DiagnosticCode string

const (
	InvalidManifest          DiagnosticCode = "invalid_manifest"
	ManifestLoadFailed       DiagnosticCode = "manifest_load_failed"
	InvalidTechnology        DiagnosticCode = "invalid_technology"
	IncompatibleTechnologies DiagnosticCode = "incompatible_technologies"
	MissingOperationProvider DiagnosticCode = "missing_operation_provider"
	InvalidOperation         DiagnosticCode = "invalid_operation"
	InvalidDependency        DiagnosticCode = "invalid_dependency"
	DependencyCycle          DiagnosticCode = "dependency_cycle"
)

// Plan separates successful construction from execution availability.
// Collection fields are emitted as JSON arrays, including when empty.
type Plan struct {
	ProjectName     string       `json:"projectName"`
	ManifestVersion string       `json:"manifestVersion"`
	Technologies    []Technology `json:"technologies"`
	Operations      []Operation  `json:"operations"`
	Diagnostics     []Diagnostic `json:"diagnostics"`
	Status          Status       `json:"status"`
	Executable      bool         `json:"executable"`
}

type Technology struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Category catalog.Category `json:"category"`
}

type Operation struct {
	ID           string          `json:"id"`
	Description  string          `json:"description"`
	Type         OperationType   `json:"type"`
	Technology   string          `json:"technology,omitempty"`
	PluginID     string          `json:"pluginId,omitempty"`
	Source       string          `json:"source,omitempty"`
	SourceType   string          `json:"sourceType,omitempty"`
	SourceDigest string          `json:"sourceDigest,omitempty"`
	TargetPaths  []string        `json:"targetPaths"`
	Dependencies []string        `json:"dependencies"`
	Status       OperationStatus `json:"status"`
}

type Diagnostic struct {
	Severity    Severity       `json:"severity"`
	Code        DiagnosticCode `json:"code"`
	Message     string         `json:"message"`
	OperationID string         `json:"operationId,omitempty"`
	Technology  string         `json:"technology,omitempty"`
	Field       string         `json:"field,omitempty"`
}
