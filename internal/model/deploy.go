package model

import "time"

type DeployStatus string

const (
	StatusRunning DeployStatus = "running"
	StatusSuccess DeployStatus = "success"
	StatusFailed  DeployStatus = "failed"
)

type DeployRecord struct {
	ID        string       `json:"id"`
	Project   string       `json:"project"`
	CommitSHA string       `json:"commit_sha"`
	Branch    string       `json:"branch"`
	Pusher    string       `json:"pusher"`
	Status    DeployStatus `json:"status"`
	Output    string       `json:"output"`
	StartedAt time.Time    `json:"started_at"`
	EndedAt   *time.Time   `json:"ended_at,omitempty"`
}

type Project struct {
	Name   string
	Secret string
	Script string
	Branch string
}

type GitHubPushPayload struct {
	Ref    string `json:"ref"`
	After  string `json:"after"`
	Pusher struct {
		Name string `json:"name"`
	} `json:"pusher"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}
