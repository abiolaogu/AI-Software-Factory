package domain

type ErrorDetail struct {
	Field string `json:"field,omitempty"`
	Error string `json:"error,omitempty"`
}

type ErrorResponse struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type,omitempty"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type Pagination struct {
	Page  int `json:"page,omitempty"`
	Size  int `json:"size,omitempty"`
	Total int `json:"total,omitempty"`
	Pages int `json:"pages,omitempty"`
}

type WorkflowStage string

const (
	WorkflowStageInit          WorkflowStage = "init"
	WorkflowStagePRD           WorkflowStage = "prd"
	WorkflowStageDesign        WorkflowStage = "design"
	WorkflowStageAccessibility WorkflowStage = "a11y"
	WorkflowStageArchitecture  WorkflowStage = "architecture"
	WorkflowStagePlan          WorkflowStage = "plan"
	WorkflowStageCode          WorkflowStage = "code"
	WorkflowStageTest          WorkflowStage = "test"
	WorkflowStageSecurity      WorkflowStage = "security"
	WorkflowStagePerformance   WorkflowStage = "performance"
	WorkflowStagePackage       WorkflowStage = "package"
	WorkflowStageDeploy        WorkflowStage = "deploy"
	WorkflowStageEval          WorkflowStage = "eval"
	WorkflowStageDone          WorkflowStage = "done"
	WorkflowStageFailed        WorkflowStage = "failed"
)

type GitStrategy string

const (
	GitStrategyTrunkBased GitStrategy = "trunk-based"
	GitStrategyGitFlow    GitStrategy = "gitflow"
)

type WorkflowRequest struct {
	Prompt          string         `json:"prompt"`
	Domain          string         `json:"domain,omitempty"`
	GitStrategy     GitStrategy    `json:"git_strategy,omitempty"`
	TargetLanguages []string       `json:"target_languages,omitempty"`
	Regions         []string       `json:"regions,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type WorkflowLinks struct {
	Self      string `json:"self,omitempty"`
	Artifacts string `json:"artifacts,omitempty"`
	Cancel    string `json:"cancel,omitempty"`
}

type WorkflowResponse struct {
	WorkflowID          string            `json:"workflow_id"`
	Status              string            `json:"status"`
	CurrentStage        WorkflowStage     `json:"current_stage"`
	Progress            float64           `json:"progress"`
	CreatedAt           string            `json:"created_at"`
	StartedAt           string            `json:"started_at,omitempty"`
	CompletedAt         string            `json:"completed_at,omitempty"`
	EstimatedCompletion string            `json:"estimated_completion,omitempty"`
	Artifacts           map[string]string `json:"artifacts,omitempty"`
	Errors              []string          `json:"errors,omitempty"`
	Links               WorkflowLinks     `json:"_links,omitempty"`
}

type WorkflowListResponse struct {
	Workflows  []WorkflowResponse `json:"workflows"`
	Pagination Pagination         `json:"pagination"`
}

type ArtifactDescriptor struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	Size      int    `json:"size"`
	CreatedAt string `json:"created_at"`
}

type WorkflowArtifactsResponse struct {
	Artifacts map[string]ArtifactDescriptor `json:"artifacts"`
}

type PRDRequest struct {
	Prompt        string   `json:"prompt"`
	Domain        string   `json:"domain,omitempty"`
	ReferencePRDs []string `json:"reference_prds,omitempty"`
}

type SuccessMetric struct {
	Name        string `json:"name"`
	Target      string `json:"target"`
	Measurement string `json:"measurement"`
}

type PRDResponse struct {
	PRDID                 string          `json:"prd_id"`
	Markdown              string          `json:"markdown"`
	JSON                  map[string]any  `json:"json,omitempty"`
	SuccessMetrics        []SuccessMetric `json:"success_metrics,omitempty"`
	UserStories           []string        `json:"user_stories,omitempty"`
	TechnicalRequirements []string        `json:"technical_requirements,omitempty"`
	CreatedAt             string          `json:"created_at"`
}

type DesignRequest struct {
	PRDID       string `json:"prd_id"`
	Style       string `json:"style,omitempty"`
	ColorScheme string `json:"color_scheme,omitempty"`
}

type DesignAsset struct {
	ScreenName string `json:"screen_name"`
	URL        string `json:"url"`
}

type DesignTokens struct {
	Colors     map[string]string `json:"colors,omitempty"`
	Typography map[string]string `json:"typography,omitempty"`
	Spacing    map[string]string `json:"spacing,omitempty"`
}

type DesignResponse struct {
	DesignID    string        `json:"design_id"`
	Wireframes  []DesignAsset `json:"wireframes,omitempty"`
	HifiDesigns []DesignAsset `json:"hifi_designs,omitempty"`
	Tokens      DesignTokens  `json:"tokens,omitempty"`
}

type DesignArtifacts struct {
	Wireframes  []DesignAsset `json:"wireframes,omitempty"`
	HifiDesigns []DesignAsset `json:"hifi_designs,omitempty"`
}

type ExtractTokensRequest struct {
	DesignID string `json:"design_id"`
}

type ExtractTokensResponse struct {
	Tokens map[string]any `json:"tokens"`
}

type CodeSpecification struct {
	Description  string   `json:"description,omitempty"`
	Interface    string   `json:"interface,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
}

type CodeRequest struct {
	Specification CodeSpecification `json:"specification"`
	Language      string            `json:"language"`
	Framework     string            `json:"framework,omitempty"`
}

type CodeValidation struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

type CodeResponse struct {
	CodeID     string         `json:"code_id"`
	Code       string         `json:"code"`
	Language   string         `json:"language"`
	FilePath   string         `json:"file_path"`
	Tests      string         `json:"tests,omitempty"`
	Validation CodeValidation `json:"validation"`
}

type GenerateTestsRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type GenerateTestsResponse struct {
	Tests    string  `json:"tests"`
	Coverage float64 `json:"coverage"`
}

type SecurityScanRequest struct {
	Target    string   `json:"target"`
	ScanTypes []string `json:"scan_types,omitempty"`
}

type SecurityFindingLocation struct {
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

type SecurityFinding struct {
	FindingID   string                  `json:"finding_id,omitempty"`
	Severity    string                  `json:"severity,omitempty"`
	Category    string                  `json:"category,omitempty"`
	Description string                  `json:"description,omitempty"`
	Location    SecurityFindingLocation `json:"location,omitempty"`
	Remediation string                  `json:"remediation,omitempty"`
	CWE         []string                `json:"cwe,omitempty"`
	CVSSScore   float64                 `json:"cvss_score,omitempty"`
}

type SecuritySummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}

type SecurityScanQueuedResponse struct {
	ScanID string `json:"scan_id"`
	Status string `json:"status"`
}

type SecurityScanResponse struct {
	ScanID    string            `json:"scan_id"`
	Status    string            `json:"status"`
	Findings  []SecurityFinding `json:"findings,omitempty"`
	Summary   SecuritySummary   `json:"summary"`
	SBOM      string            `json:"sbom,omitempty"`
	Signature string            `json:"signature,omitempty"`
}

type SecurityEvidence struct {
	SBOM      string `json:"sbom"`
	Signature string `json:"signature"`
}

type PerformanceTestRequest struct {
	Endpoint     string   `json:"endpoint"`
	Duration     string   `json:"duration,omitempty"`
	VirtualUsers int      `json:"virtual_users,omitempty"`
	Regions      []string `json:"regions,omitempty"`
}

type ThresholdResult struct {
	Metric    string  `json:"metric"`
	Threshold float64 `json:"threshold"`
	Actual    float64 `json:"actual"`
	Passed    bool    `json:"passed"`
}

type PerformanceMetrics struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	LatencyP50        float64 `json:"latency_p50"`
	LatencyP95        float64 `json:"latency_p95"`
	LatencyP99        float64 `json:"latency_p99"`
	ErrorRate         float64 `json:"error_rate"`
}

type PerformanceStartResponse struct {
	TestID string `json:"test_id"`
}

type PerformanceTestResponse struct {
	TestID     string             `json:"test_id"`
	Status     string             `json:"status"`
	Metrics    PerformanceMetrics `json:"metrics"`
	Passed     bool               `json:"passed"`
	Thresholds []ThresholdResult  `json:"thresholds,omitempty"`
}

type HealthLiveResponse struct {
	Status string `json:"status"`
}

type HealthReadyResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}
