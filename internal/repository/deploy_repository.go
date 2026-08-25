package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"webhook-deploy/internal/model"
)

type DeployRepository interface {
	Save(record model.DeployRecord) error
	Update(record model.DeployRecord) error
	FindAll() ([]model.DeployRecord, error)
	FindLatest(limit int) ([]model.DeployRecord, error)
}

type jsonFileDeployRepository struct {
	filePath string
	mu       sync.Mutex
}

// NewJSONFileDeployRepository creates the repository and ensures the
// backing file exists.
func NewJSONFileDeployRepository(filePath string) (DeployRepository, error) {
	repo := &jsonFileDeployRepository{filePath: filePath}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := repo.writeAll([]model.DeployRecord{}); err != nil {
			return nil, fmt.Errorf("init deploy history file: %w", err)
		}
	}
	return repo, nil
}

func (r *jsonFileDeployRepository) Save(record model.DeployRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	records, err := r.readAll()
	if err != nil {
		return err
	}
	records = append(records, record)
	return r.writeAll(records)
}

// Update finds a record by ID and replaces it (used to flip
// "running" -> "success"/"failed" once the deploy script finishes).
func (r *jsonFileDeployRepository) Update(record model.DeployRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	records, err := r.readAll()
	if err != nil {
		return err
	}

	found := false
	for i, rec := range records {
		if rec.ID == record.ID {
			records[i] = record
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("deploy record %s not found", record.ID)
	}
	return r.writeAll(records)
}

func (r *jsonFileDeployRepository) FindAll() ([]model.DeployRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.readAll()
}

func (r *jsonFileDeployRepository) FindLatest(limit int) ([]model.DeployRecord, error) {
	records, err := r.FindAll()
	if err != nil {
		return nil, err
	}

	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}
	if limit > 0 && limit < len(records) {
		records = records[:limit]
	}

	return records, nil
}

// readAll / writeAll are the only two places that touch the filesystem —
// callers must hold r.mu before calling these.
func (r *jsonFileDeployRepository) readAll() ([]model.DeployRecord, error) {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, fmt.Errorf("read deploy history: %w", err)
	}

	var records []model.DeployRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("parse deploy history: %w", err)
	}

	return records, nil
}

func (r *jsonFileDeployRepository) writeAll(records []model.DeployRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal deploy history: %w", err)
	}

	if err := os.WriteFile(r.filePath, data, 0644); err != nil {
		return fmt.Errorf("write deploy history: %w", err)
	}
	return nil
}
