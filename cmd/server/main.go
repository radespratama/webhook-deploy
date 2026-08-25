package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"webhook-deploy/internal/config"
	"webhook-deploy/internal/controller"
	"webhook-deploy/internal/middleware"
	"webhook-deploy/internal/repository"
	"webhook-deploy/internal/router"
	"webhook-deploy/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	deployRepo, err := repository.NewJSONFileDeployRepository(cfg.DeployHistoryFP)
	if err != nil {
		log.Fatalf("repository error: %v", err)
	}

	githubService := service.NewGitHubService()
	deployService := service.NewDeployService(deployRepo, cfg.DiscordWebhookURL, cfg.ErrorLogDir)

	webhookController := controller.NewWebhookController(githubService, deployService, cfg)

	app := fiber.New(fiber.Config{
		AppName: "webhook-deploy",
	})

	app.Use(middleware.RequestLogger())

	router.Setup(app, webhookController)

	log.Printf("listening on :%s (%d project(s))", cfg.Port, len(cfg.Projects))
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
