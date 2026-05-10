package deploymentstore

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

func (s *Store) Create(ctx context.Context, deployment *models.Deployment) error {
	if err := s.db.WithContext(ctx).Create(deployment).Error; err != nil {
		return fmt.Errorf("create deployment: %w", err)
	}

	return nil
}

func (s *Store) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	var deployment models.Deployment

	err := s.db.WithContext(ctx).First(&deployment, "id = ?", id).Error
	if err != nil {
		return nil, fmt.Errorf("get deployment %q: %w", id, err)
	}

	return &deployment, nil
}

func (s *Store) List(ctx context.Context) ([]models.Deployment, error) {
	var deployments []models.Deployment

	err := s.db.WithContext(ctx).
		Order("created_at DESC").
		Find(&deployments).Error
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}

	return deployments, nil
}

func (s *Store) Update(ctx context.Context, deployment *models.Deployment) error {
	if err := s.db.WithContext(ctx).Model(deployment).Updates(deployment).Error; err != nil {
		return fmt.Errorf("update deployment %q: %w", deployment.ID, err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	if err := s.db.WithContext(ctx).Delete(&models.Deployment{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("delete deployment %q: %w", id, err)
	}

	return nil
}
