package config

import (
	"fmt"
	"os"
	"strings"

	"webhook-deploy/internal/model"
)

type Config struct {
	Port              string
	Projects          []model.Project
	DeployHistoryFP   string
	DiscordWebhookURL string
	ErrorLogDir       string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnv("PORT", "9000"),
		DeployHistoryFP:   getEnv("DEPLOY_HISTORY_PATH", "./deploy_history.json"),
		DiscordWebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),
		ErrorLogDir:       getEnv("ERROR_LOG_DIR", "./logs"),
	}

	for _, raw := range strings.Split(getEnv("PROJECTS", "kuroneko"), ",") {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		envKey := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))

		secret := os.Getenv(envKey + "_WEBHOOK_SECRET")
		if secret == "" {
			return nil, fmt.Errorf("%s_WEBHOOK_SECRET is required but not set", envKey)
		}

		cfg.Projects = append(cfg.Projects, model.Project{
			Name:   name,
			Secret: secret,
			Script: getEnv(envKey+"_DEPLOY_SCRIPT", fmt.Sprintf("./deploys/%s.deploy.sh", name)),
			Branch: getEnv(envKey+"_ALLOWED_BRANCH", "refs/heads/main"),
		})
	}

	if len(cfg.Projects) == 0 {
		return nil, fmt.Errorf("PROJECTS is required but not set or empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
