package artifacts

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/stratogroup/neuroforge-sdx/internal/domain"
)

type SignedArtifactService struct {
	serviceName string
	baseURL     *url.URL
	signingKey  []byte
	ttl         time.Duration
}

func NewSignedArtifactService(serviceName, baseURL, signingKey string, ttl time.Duration) (*SignedArtifactService, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse artifact base url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("artifact base url must include scheme and host")
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	key := strings.TrimSpace(signingKey)
	if key == "" {
		key = "neuroforge-sdx-local-signing-key"
	}

	return &SignedArtifactService{
		serviceName: strings.TrimSpace(serviceName),
		baseURL:     parsed,
		signingKey:  []byte(key),
		ttl:         ttl,
	}, nil
}

func (s *SignedArtifactService) WorkflowArtifacts(_ context.Context, tenantID, workflowID string, now time.Time) (map[string]domain.ArtifactDescriptor, error) {
	manifestURL, err := s.signedObjectURL(tenantID, now, "workflows", workflowID, "manifest.json")
	if err != nil {
		return nil, err
	}
	openapiURL, err := s.signedObjectURL(tenantID, now, "workflows", workflowID, "openapi.json")
	if err != nil {
		return nil, err
	}

	createdAt := now.UTC().Format(time.RFC3339)
	return map[string]domain.ArtifactDescriptor{
		"manifest": {
			URL:       manifestURL,
			Type:      "application/json",
			Size:      2048,
			CreatedAt: createdAt,
		},
		"openapi": {
			URL:       openapiURL,
			Type:      "application/json",
			Size:      4096,
			CreatedAt: createdAt,
		},
	}, nil
}

func (s *SignedArtifactService) DesignArtifacts(_ context.Context, tenantID, designID string, now time.Time) (domain.DesignArtifacts, error) {
	wireframeURL, err := s.signedObjectURL(tenantID, now, "designs", designID, "workflow-dashboard.png")
	if err != nil {
		return domain.DesignArtifacts{}, err
	}
	viewerURL, err := s.signedObjectURL(tenantID, now, "designs", designID, "artifact-viewer.png")
	if err != nil {
		return domain.DesignArtifacts{}, err
	}
	hifiURL, err := s.signedObjectURL(tenantID, now, "designs", designID, "workflow-dashboard-hifi.png")
	if err != nil {
		return domain.DesignArtifacts{}, err
	}

	return domain.DesignArtifacts{
		Wireframes: []domain.DesignAsset{
			{ScreenName: "workflow-dashboard", URL: wireframeURL},
			{ScreenName: "artifact-viewer", URL: viewerURL},
		},
		HifiDesigns: []domain.DesignAsset{
			{ScreenName: "workflow-dashboard", URL: hifiURL},
		},
	}, nil
}

func (s *SignedArtifactService) SecurityEvidence(_ context.Context, tenantID, scanID string, now time.Time) (domain.SecurityEvidence, error) {
	sbomURL, err := s.signedObjectURL(tenantID, now, "compliance", scanID, "sbom.spdx.json")
	if err != nil {
		return domain.SecurityEvidence{}, err
	}

	expiresAt := now.UTC().Add(s.ttl).Unix()
	signature := s.signPayload(tenantID, path.Join("compliance", scanID, "signature"), expiresAt)
	return domain.SecurityEvidence{
		SBOM:      sbomURL,
		Signature: fmt.Sprintf("cosign://rekor/%s/%s?tenant=%s&expires=%d&signature=%s", s.serviceName, scanID, tenantID, expiresAt, signature),
	}, nil
}

func (s *SignedArtifactService) signedObjectURL(tenantID string, now time.Time, segments ...string) (string, error) {
	signed := *s.baseURL
	objectPath := path.Join(segments...)
	signed.Path = joinURLPath(s.baseURL.Path, objectPath)

	expiresAt := now.UTC().Add(s.ttl).Unix()
	query := signed.Query()
	query.Set("tenant", tenantID)
	query.Set("expires", strconv.FormatInt(expiresAt, 10))
	query.Set("signature", s.signPayload(tenantID, objectPath, expiresAt))
	signed.RawQuery = query.Encode()

	return signed.String(), nil
}

func (s *SignedArtifactService) signPayload(tenantID, objectPath string, expiresAt int64) string {
	mac := hmac.New(sha256.New, s.signingKey)
	_, _ = mac.Write([]byte(tenantID))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(objectPath))
	_, _ = mac.Write([]byte{'\n'})
	_, _ = mac.Write([]byte(strconv.FormatInt(expiresAt, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

func joinURLPath(basePath, objectPath string) string {
	joined := path.Join("/", strings.TrimSpace(basePath), objectPath)
	if !strings.HasPrefix(joined, "/") {
		return "/" + joined
	}
	return joined
}
