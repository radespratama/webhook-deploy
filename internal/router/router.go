package router

import (
	"github.com/gofiber/fiber/v2"

	"webhook-deploy/internal/controller"
)

func Setup(app *fiber.App, wc *controller.WebhookController) {
	app.Get("/health", wc.HandleHealth)

	app.Post("/github", wc.HandleGitHubWebhook)

	app.Get("/deploys", wc.HandleDeployHistory)
}
