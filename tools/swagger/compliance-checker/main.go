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
	Timestamp     string           `json:"timestamp"`
	SpecFile      string           `json:"spec_file"`
	Status        string           `json:"status"` // PASS or FAIL
	TotalEndpoints int             `json:"total_endpoints"`
	ValidEndpoints int             `json:"valid_endpoints"`
	InvalidEndpoints int           `json:"invalid_endpoints"`
	Checks        []CheckResult    `json:"checks"`
	Violations    []Violation      `json:"violations"`
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
	specFile := "../../api/openapi/sdx-gateway-v1.swagger.json"
	if len(os.Args) > 1 {
		specFile = os.Args[1]
	}

	data, err := os.ReadFile(specFile)
	if err != nil {
		log.Fatalf("Failed to read spec file: %v", err)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		log.Fatalf("Failed to parse spec: %v", err)
	}

	report := &ComplianceReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		SpecFile:  specFile,
	}

	// Run all checks.
	report.Checks = append(report.Checks, checkInfo(spec))
	report.Checks = append(report.Checks, checkSecurityDefinitions(spec))
	endpointChecks, violations, total, valid := checkEndpoints(spec)
	report.Checks = append(report.Checks, endpointChecks...)
	report.Violations = violations
	report.TotalEndpoints = total
	report.ValidEndpoints = valid
	report.InvalidEndpoints = total - valid
	report.Checks = append(report.Checks, checkPaths(spec))

	// Determine overall status.
	hasError := false
	for _, v := range report.Violations {
		if v.Severity == "ERROR" {
			hasError = true
			break
		}
	}
	for _, c := range report.Checks {
		if c.Status == "FAIL" {
			hasError = true
			break
		}
	}

	if hasError {
		report.Status = "FAIL"
	} else {
		report.Status = "PASS"
	}

	// Write report.
	reportData, _ := json.MarshalIndent(report, "", "  ")

	reportFile := "compliance-report.json"
	if len(os.Args) > 2 {
		reportFile = os.Args[2]
	}
	if err := os.WriteFile(reportFile, reportData, 0644); err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}

	fmt.Printf("Compliance Report: %s\n", report.Status)
	fmt.Printf("  Total endpoints: %d\n", report.TotalEndpoints)
	fmt.Printf("  Valid:           %d\n", report.ValidEndpoints)
	fmt.Printf("  Invalid:         %d\n", report.InvalidEndpoints)
	fmt.Printf("  Violations:      %d\n", len(report.Violations))
	fmt.Printf("  Report written:  %s\n", reportFile)

	if report.Status == "FAIL" {
		for _, v := range report.Violations {
			fmt.Printf("  [%s] %s %s: %s — %s\n", v.Severity, v.Method, v.Path, v.Rule, v.Message)
		}
		os.Exit(1)
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
			endpointValid := true

			// Check operationId.
			opId, _ := op["operationId"].(string)
			if opId == "" {
				violations = append(violations, Violation{
					Path: path, Method: strings.ToUpper(method),
					Rule: "operation-id-required", Message: "Missing operationId",
					Severity: "ERROR",
				})
				endpointValid = false
			}

			// Check tags.
			tags, _ := op["tags"].([]interface{})
			if len(tags) == 0 {
				violations = append(violations, Violation{
					Path: path, Method: strings.ToUpper(method),
					Rule: "tags-required", Message: "Missing tags for API grouping",
					Severity: "WARNING",
				})
			}

			// Check responses.
			responses, _ := op["responses"].(map[string]interface{})
			if len(responses) == 0 {
				violations = append(violations, Violation{
					Path: path, Method: strings.ToUpper(method),
					Rule: "responses-required", Message: "No response definitions",
					Severity: "ERROR",
				})
				endpointValid = false
			}

			// Check for success response (200 or default).
			if len(responses) > 0 {
				_, has200 := responses["200"]
				_, hasDefault := responses["default"]
				if !has200 && !hasDefault {
					violations = append(violations, Violation{
						Path: path, Method: strings.ToUpper(method),
						Rule: "success-response-required", Message: "No 200 or default response defined",
						Severity: "WARNING",
					})
				}
			}

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
