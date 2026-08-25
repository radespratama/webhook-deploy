package controller

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"webhook-deploy/internal/config"
	"webhook-deploy/internal/model"
	"webhook-deploy/internal/service"
)

type WebhookController struct {
	githubService service.GitHubService
	deployService service.DeployService
	cfg           *config.Config
}

func NewWebhookController(gs service.GitHubService, ds service.DeployService, cfg *config.Config) *WebhookController {
	return &WebhookController{
		githubService: gs,
		deployService: ds,
		cfg:           cfg,
	}
}

func (wc *WebhookController) HandleGitHubWebhook(c *fiber.Ctx) error {
	rawBody := c.Body()

	signature := c.Get("X-Hub-Signature-256")
	if signature == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing X-Hub-Signature-256 header",
		})
	}

	project, ok := wc.matchProject(rawBody, signature)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid signature",
		})
	}

	event := c.Get("X-GitHub-Event")
	if event == "ping" {
		return c.JSON(fiber.Map{"message": "pong", "project": project.Name})
	}

	if event != "push" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "only push events are supported, got: " + event,
		})
	}

	push, err := wc.githubService.ParsePushPayload(rawBody)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "malformed payload: " + err.Error(),
		})
	}

	if push.Ref != project.Branch {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "ignored: branch " + push.Ref + " is not " + project.Branch,
		})
	}

	deployID, err := wc.deployService.TriggerDeploy(project, push)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to start deploy: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message":   "deploy triggered",
		"project":   project.Name,
		"deploy_id": deployID,
	})
}

func (wc *WebhookController) matchProject(rawBody []byte, signature string) (model.Project, bool) {
	for _, p := range wc.cfg.Projects {
		if wc.githubService.VerifySignature(rawBody, signature, p.Secret) {
			return p, true
		}
	}
	return model.Project{}, false
}

func (wc *WebhookController) HandleDeployHistory(c *fiber.Ctx) error {
	limit := 20
	if q := c.Query("limit"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil {
			limit = parsed
		}
	}

	records, err := wc.deployService.History(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(records)
}

func (wc *WebhookController) HandleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}
