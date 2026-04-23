package models

import "time"

// —— Deployment ————————————————————————————————————————————————————————————————————————————————

type DeploymentStatus string

const (
	StatusPending   DeploymentStatus = "pending"
	StatusBuilding  DeploymentStatus = "building"
	StatusDeploying DeploymentStatus = "deploying"
	StatusRunning   DeploymentStatus = "running"
	StatusFailed    DeploymentStatus = "failed"
)

type Deployment struct {
	ID            string           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	GithubURL     string           `gorm:"not null" json:"github_url"`
	Status        DeploymentStatus `gorm:"default:pending" json:"status"`
	ImageTag      string           `json:"image_tag,omitempty"`
	ContainerPort int              `json:"container_port,omitempty"`
	ContainerID   string           `json:"container_id,omitempty"`
	LiveURL       string           `json:"live_url,omitempty"`
	CreatedAt     time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Deployment) TableName() string { return "deployments" }

// —— Logs ———————————————————————————————————————————————————————————————————————————————

type LogEntry struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeploymentID string    `gorm:"type:uuid;not null;index" json:"deployment_id"`
	Message      string    `gorm:"not null" json:"message"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`

	Deployment *Deployment `gorm:"foreignKey:DeploymentID;constraint:OnDelete:CASCADE" json:"-"`
}

func (LogEntry) TableName() string { return "logs" }
