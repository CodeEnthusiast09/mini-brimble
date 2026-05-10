package logstore

import (
	"context"
	"fmt"

	"github.com/CodeEnthusiast09/mini-brimble/server/internal/models"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Save(ctx context.Context, deploymentID string, message string) (*models.LogEntry, error) {
	entry := models.LogEntry{
		DeploymentID: deploymentID,
		Message:      message,
	}

	if err := s.db.WithContext(ctx).Create(&entry).Error; err != nil {
		return nil, fmt.Errorf("save log entry: %w", err)
	}

	return &entry, nil
}

func (s *Store) GetByDeploymentID(ctx context.Context, deploymentID string) ([]models.LogEntry, error) {
	var entries []models.LogEntry

	err := s.db.WithContext(ctx).
		Where("deployment_id = ?", deploymentID).
		Order("created_at ASC").
		Find(&entries).Error
	if err != nil {
		return nil, fmt.Errorf("get logs for deployment %q: %w", deploymentID, err)
	}

	return entries, nil
}
