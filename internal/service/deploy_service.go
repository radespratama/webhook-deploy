package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"

	"webhook-deploy/internal/model"
	"webhook-deploy/internal/notifier"
	"webhook-deploy/internal/repository"
)

type DeployService interface {
	TriggerDeploy(project model.Project, push model.GitHubPushPayload) (string, error)

	History(limit int) ([]model.DeployRecord, error)
}

type deployService struct {
	repo              repository.DeployRepository
	scriptTimeout     time.Duration
	discordWebhookURL string
	errorLogDir       string
}

func NewDeployService(
	repo repository.DeployRepository,
	discordWebhookURL,
	errorLogDir string,
) DeployService {
	return &deployService{
		repo:              repo,
		scriptTimeout:     10 * time.Minute,
		discordWebhookURL: discordWebhookURL,
		errorLogDir:       errorLogDir,
	}
}

func (d *deployService) TriggerDeploy(
	project model.Project,
	push model.GitHubPushPayload,
) (string, error) {
	record := model.DeployRecord{
		ID:        uuid.NewString(),
		Project:   project.Name,
		CommitSHA: push.After,
		Branch:    push.Ref,
		Pusher:    push.Pusher.Name,
		Status:    model.StatusRunning,
		StartedAt: time.Now(),
	}

	if err := d.repo.Save(record); err != nil {
		return "", fmt.Errorf("save initial deploy record: %w", err)
	}

	go d.runDeploy(project, record)

	return record.ID, nil
}

func (d *deployService) runDeploy(
	project model.Project,
	record model.DeployRecord,
) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		d.scriptTimeout,
	)
	defer cancel()

	startedAt := time.Now()

	cmd := exec.CommandContext(
		ctx,
		"/bin/bash",
		project.Script,
	)

	cmd.Env = append(
		os.Environ(),
		"DEPLOY_PROJECT="+project.Name,
	)

	output, err := cmd.CombinedOutput()

	now := time.Now()
	record.EndedAt = &now
	record.Output = string(output)

	duration := time.Since(startedAt).Round(time.Millisecond)

	// ============================================================
	// DEPLOY FAILED
	// ============================================================

	if err != nil {
		record.Status = model.StatusFailed
		record.Output += fmt.Sprintf(
			"\n--- error: %v ---",
			err,
		)

		log.Printf(
			"[deploy %s] FAILED: %v",
			record.ID,
			err,
		)

		notifier.AppendError(
			d.errorLogDir,
			fmt.Sprintf(
				"[deploy %s] project=%s branch=%s commit=%s pusher=%s failed: %v",
				record.ID,
				project.Name,
				record.Branch,
				record.CommitSHA,
				record.Pusher,
				err,
			),
		)

		message := fmt.Sprintf(
			`💀 **Deploying to %s FAILED**
Yah... **deploy gagal** 😵

📦 **Project:** %s
🔖 **Commit:** %s
👤 **Pusher:** %s
⏱️ **Duration:** %s

`+"```"+`
%v
`+"```",
			record.Branch,
			project.Name,
			record.CommitSHA,
			record.Pusher,
			duration,
			err,
		)

		notifier.Discord(
			d.discordWebhookURL,
			message,
			"https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExbWdmNDFvbHYxdHJmN2gwZG9nYnhhY3EyMDY0bmlnM3JwODA5d2dreSZlcD12MV9naWZzX3NlYXJjaCZjdD1n/zZC2AqB84z7zFnlkbF/giphy.gif",
		)

	} else {

		// ============================================================
		// DEPLOY SUCCESS
		// ============================================================

		record.Status = model.StatusSuccess

		log.Printf(
			"[deploy %s] success",
			record.ID,
		)

		message := fmt.Sprintf(
			`🚀 **Deploying to %s SUCCESS**
Yay !!! Projectmu udah berhasil di deploy yah 🚀

📦 **Project:** %s
🔖 **Commit:** %s
👤 **Pusher:** %s
⏱️ **Duration:** %s`,
			record.Branch,
			project.Name,
			record.CommitSHA,
			record.Pusher,
			duration,
		)

		notifier.Discord(
			d.discordWebhookURL,
			message,
			"https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExbWdmNDFvbHYxdHJmN2gwZG9nYnhhY3EyMDY0bmlnM3JwODA5d2dreSZlcD12MV9naWZzX3NlYXJjaCZjdD1n/zZC2AqB84z7zFnlkbF/giphy.gif",
		)
	}

	// ============================================================
	// SAVE FINAL DEPLOY STATUS
	// ============================================================

	if updateErr := d.repo.Update(record); updateErr != nil {
		log.Printf(
			"[deploy %s] failed to persist final status: %v",
			record.ID,
			updateErr,
		)
	}
}

func (d *deployService) History(
	limit int,
) ([]model.DeployRecord, error) {
	return d.repo.FindLatest(limit)
}
