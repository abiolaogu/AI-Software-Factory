// Package main implements a comprehensive OpenAPI compliance checker.
// It validates that every endpoint in the generated OpenAPI spec meets
// the NeuroForge SDX API standards from docs/api/.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type ComplianceReport struct {
	Timestamp        string        `json:"timestamp"`
	SpecFile         string        `json:"spec_file"`
	Status           string        `json:"status"` // PASS or FAIL
	TotalEndpoints   int           `json:"total_endpoints"`
	ValidEndpoints   int           `json:"valid_endpoints"`
	InvalidEndpoints int           `json:"invalid_endpoints"`
	Checks           []CheckResult `json:"checks"`
	Violations       []Violation   `json:"violations"`
}

type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // PASS or FAIL
	Details string `json:"details"`
}

type Violation struct {
	Path     string `json:"path"`
	Method   string `json:"method"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // ERROR, WARNING
}

func main() {
	specFile := argument(1, "../../api/openapi/sdx-gateway-v1.swagger.json")
	reportFile := argument(2, "compliance-report.json")
	spec := loadSpec(specFile)
	report := buildReport(specFile, spec)
	writeReport(reportFile, report)
	printSummary(reportFile, report)
	if report.Status == "FAIL" {
		printViolations(report.Violations)
		os.Exit(1)
	}
}

func argument(index int, fallback string) string {
	if len(os.Args) > index {
		return os.Args[index]
	}
	return fallback
}

func loadSpec(specFile string) map[string]interface{} {
	data, err := os.ReadFile(specFile)
	if err != nil {
		log.Fatalf("Failed to read spec file: %v", err)
	}
	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		log.Fatalf("Failed to parse spec: %v", err)
	}
	return spec
}

func buildReport(specFile string, spec map[string]interface{}) *ComplianceReport {
	report := &ComplianceReport{Timestamp: time.Now().UTC().Format(time.RFC3339), SpecFile: specFile}
	report.Checks = append(report.Checks, checkInfo(spec), checkSecurityDefinitions(spec))
	endpointChecks, violations, total, valid := checkEndpoints(spec)
	report.Checks = append(report.Checks, endpointChecks...)
	report.Checks = append(report.Checks, checkPaths(spec))
	report.Violations = violations
	report.TotalEndpoints = total
	report.ValidEndpoints = valid
	report.InvalidEndpoints = total - valid
	report.Status = complianceStatus(report)
	return report
}

func complianceStatus(report *ComplianceReport) string {
	for _, violation := range report.Violations {
		if violation.Severity == "ERROR" {
			return "FAIL"
		}
	}
	for _, check := range report.Checks {
		if check.Status == "FAIL" {
			return "FAIL"
		}
	}
	return "PASS"
}

func writeReport(reportFile string, report *ComplianceReport) {
	reportData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("Failed to encode report: %v", err)
	}
	if err := os.WriteFile(reportFile, reportData, 0644); err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}
}

func printSummary(reportFile string, report *ComplianceReport) {
	fmt.Printf("Compliance Report: %s\n", report.Status)
	fmt.Printf("  Total endpoints: %d\n", report.TotalEndpoints)
	fmt.Printf("  Valid:           %d\n", report.ValidEndpoints)
	fmt.Printf("  Invalid:         %d\n", report.InvalidEndpoints)
	fmt.Printf("  Violations:      %d\n", len(report.Violations))
	fmt.Printf("  Report written:  %s\n", reportFile)
}

func printViolations(violations []Violation) {
	for _, violation := range violations {
		fmt.Printf("  [%s] %s %s: %s — %s\n", violation.Severity, violation.Method, violation.Path, violation.Rule, violation.Message)
	}
}

// checkInfo validates the info block has proper title, version, and description.
func checkInfo(spec map[string]interface{}) CheckResult {
	info, ok := spec["info"].(map[string]interface{})
	if !ok {
		return CheckResult{Name: "info-block", Status: "FAIL", Details: "Missing info block"}
	}

	issues := []string{}
	title, _ := info["title"].(string)
	if title == "" || strings.HasSuffix(title, ".proto") {
		issues = append(issues, "title is empty or a proto file path")
	}

	version, _ := info["version"].(string)
	if version == "" || version == "version not set" {
		issues = append(issues, "version is not set")
	}

	desc, _ := info["description"].(string)
	if desc == "" {
		issues = append(issues, "description is empty")
	}

	if len(issues) > 0 {
		return CheckResult{Name: "info-block", Status: "FAIL", Details: strings.Join(issues, "; ")}
	}
	return CheckResult{Name: "info-block", Status: "PASS", Details: fmt.Sprintf("title=%q version=%q", title, version)}
}

// checkSecurityDefinitions validates auth security definitions exist.
func checkSecurityDefinitions(spec map[string]interface{}) CheckResult {
	secDefs, ok := spec["securityDefinitions"].(map[string]interface{})
	if !ok || len(secDefs) == 0 {
		return CheckResult{Name: "security-definitions", Status: "FAIL", Details: "No security definitions found"}
	}

	names := make([]string, 0, len(secDefs))
	for name := range secDefs {
		names = append(names, name)
	}

	return CheckResult{
		Name:    "security-definitions",
		Status:  "PASS",
		Details: fmt.Sprintf("Found %d definitions: %s", len(secDefs), strings.Join(names, ", ")),
	}
}

// checkEndpoints validates each endpoint has operationId, tags, and responses.
func checkEndpoints(spec map[string]interface{}) ([]CheckResult, []Violation, int, int) {
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok || len(paths) == 0 {
		return []CheckResult{
			{Name: "endpoints", Status: "FAIL", Details: "No paths defined in spec"},
		}, nil, 0, 0
	}

	methods := []string{"get", "post", "put", "patch", "delete"}
	total := 0
	valid := 0
	var violations []Violation

	for path, pathItem := range paths {
		pathObj, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}
		for _, method := range methods {
			op, ok := pathObj[method].(map[string]interface{})
			if !ok {
				continue
			}
			total++
			endpointViolations, endpointValid := checkEndpoint(path, method, op)
			violations = append(violations, endpointViolations...)
			if endpointValid {
				valid++
			}
		}
	}

	status := "PASS"
	if valid < total {
		status = "FAIL"
	}

	return []CheckResult{
		{Name: "endpoint-compliance", Status: status, Details: fmt.Sprintf("%d/%d endpoints compliant", valid, total)},
	}, violations, total, valid
}

func checkEndpoint(path, method string, operation map[string]interface{}) ([]Violation, bool) {
	upperMethod := strings.ToUpper(method)
	violations := make([]Violation, 0, 3)
	valid := true
	operationID, _ := operation["operationId"].(string)
	if operationID == "" {
		violations = append(violations, Violation{Path: path, Method: upperMethod, Rule: "operation-id-required", Message: "Missing operationId", Severity: "ERROR"})
		valid = false
	}
	tags, _ := operation["tags"].([]interface{})
	if len(tags) == 0 {
		violations = append(violations, Violation{Path: path, Method: upperMethod, Rule: "tags-required", Message: "Missing tags for API grouping", Severity: "WARNING"})
	}
	responses, _ := operation["responses"].(map[string]interface{})
	if len(responses) == 0 {
		violations = append(violations, Violation{Path: path, Method: upperMethod, Rule: "responses-required", Message: "No response definitions", Severity: "ERROR"})
		valid = false
	}
	if !hasSuccessResponse(responses) {
		violations = append(violations, Violation{Path: path, Method: upperMethod, Rule: "success-response-required", Message: "No 200 or default response defined", Severity: "WARNING"})
	}
	return violations, valid
}

func hasSuccessResponse(responses map[string]interface{}) bool {
	if len(responses) == 0 {
		return true
	}
	_, hasOK := responses["200"]
	_, hasDefault := responses["default"]
	return hasOK || hasDefault
}

// checkPaths validates path naming conventions.
func checkPaths(spec map[string]interface{}) CheckResult {
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		return CheckResult{Name: "path-conventions", Status: "FAIL", Details: "No paths"}
	}

	issues := []string{}
	for path := range paths {
		if strings.HasSuffix(path, "/") {
			issues = append(issues, fmt.Sprintf("%s has trailing slash", path))
		}
		if !strings.HasPrefix(path, "/api/v1/") {
			issues = append(issues, fmt.Sprintf("%s does not start with /api/v1/", path))
		}
	}

	if len(issues) > 0 {
		return CheckResult{Name: "path-conventions", Status: "FAIL", Details: strings.Join(issues, "; ")}
	}
	return CheckResult{Name: "path-conventions", Status: "PASS", Details: fmt.Sprintf("%d paths conform to /api/v1/ convention", len(paths))}
}
