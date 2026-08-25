package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"webhook-deploy/internal/model"
)

type GitHubService interface {
	VerifySignature(payload []byte, signatureHeader string, secret string) bool
	ParsePushPayload(payload []byte) (model.GitHubPushPayload, error)
}

type gitHubService struct{}

func NewGitHubService() GitHubService {
	return &gitHubService{}
}

func (s *gitHubService) VerifySignature(payload []byte, signatureHeader string, secret string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	expectedHex := strings.TrimPrefix(signatureHeader, prefix)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	computed := mac.Sum(nil)
	computedHex := hex.EncodeToString(computed)

	return hmac.Equal([]byte(computedHex), []byte(expectedHex))
}

func (s *gitHubService) ParsePushPayload(payload []byte) (model.GitHubPushPayload, error) {
	var p model.GitHubPushPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return p, fmt.Errorf("parse github push payload: %w", err)
	}
	return p, nil
}
