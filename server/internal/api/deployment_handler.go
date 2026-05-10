package api

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/CodeEnthusiast09/mini-brimble/server/internal/deployment"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/deploymentstore"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/logstore"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/logstream"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeploymentHandler struct {
	service     *deployment.Service
	deployments *deploymentstore.Store
	logs        *logstore.Store
	streams     *logstream.Hub
}

type apiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type createDeploymentRequest struct {
	GithubURL string `json:"github_url"`
}

func NewDeploymentHandler(
	service *deployment.Service,
	deployments *deploymentstore.Store,
	logs *logstore.Store,
	streams *logstream.Hub,
) *DeploymentHandler {
	return &DeploymentHandler{
		service:     service,
		deployments: deployments,
		logs:        logs,
		streams:     streams,
	}
}

func (h *DeploymentHandler) Create(c *gin.Context) {
	var req createDeploymentRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		badRequest(c, "invalid request body")
		return
	}

	githubURL := strings.TrimSpace(req.GithubURL)
	if githubURL == "" {
		badRequest(c, "github_url is required")
		return
	}

	parsedURL, parseErr := url.ParseRequestURI(githubURL)
	if parseErr != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		badRequest(c, "github_url must be a valid http or https URL")
		return
	}

	deploymentRecord, deployErr := h.service.Deploy(c.Request.Context(), githubURL)
	if deployErr != nil {
		internalError(c, "failed to create deployment")
		return
	}

	accepted(c, "deployment accepted", deploymentRecord)
}

func (h *DeploymentHandler) List(c *gin.Context) {
	deployments, listErr := h.deployments.List(c.Request.Context())
	if listErr != nil {
		internalError(c, "failed to list deployments")
		return
	}

	ok(c, "deployments fetched successfully", deployments)
}

func (h *DeploymentHandler) Get(c *gin.Context) {
	deploymentID := c.Param("id")
	deploymentRecord, getErr := h.deployments.GetByID(c.Request.Context(), deploymentID)
	if getErr != nil {
		if errors.Is(getErr, gorm.ErrRecordNotFound) {
			notFound(c, "deployment not found")
			return
		}

		internalError(c, "failed to fetch deployment")
		return
	}

	ok(c, "deployment fetched successfully", deploymentRecord)
}

func (h *DeploymentHandler) Stop(c *gin.Context) {
	deploymentID := c.Param("id")
	stopErr := h.service.Stop(c.Request.Context(), deploymentID)
	if stopErr != nil {
		if errors.Is(stopErr, gorm.ErrRecordNotFound) {
			notFound(c, "deployment not found")
			return
		}

		internalError(c, "failed to stop deployment")
		return
	}

	ok(c, "deployment stopped successfully", nil)
}

func (h *DeploymentHandler) GetLogs(c *gin.Context) {
	deploymentID := c.Param("id")
	_, getErr := h.deployments.GetByID(c.Request.Context(), deploymentID)
	if getErr != nil {
		if errors.Is(getErr, gorm.ErrRecordNotFound) {
			notFound(c, "deployment not found")
			return
		}

		internalError(c, "failed to fetch deployment")
		return
	}

	logEntries, logsErr := h.logs.GetByDeploymentID(c.Request.Context(), deploymentID)
	if logsErr != nil {
		internalError(c, "failed to fetch deployment logs")
		return
	}

	ok(c, "deployment logs fetched successfully", logEntries)
}

func (h *DeploymentHandler) StreamLogs(c *gin.Context) {
	deploymentID := c.Param("id")
	_, getErr := h.deployments.GetByID(c.Request.Context(), deploymentID)
	if getErr != nil {
		if errors.Is(getErr, gorm.ErrRecordNotFound) {
			notFound(c, "deployment not found")
			return
		}

		internalError(c, "failed to fetch deployment")
		return
	}

	stream := h.streams.Subscribe(deploymentID)
	defer h.streams.Unsubscribe(deploymentID, stream)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		internalError(c, "streaming is not supported")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, open := <-stream:
			if !open {
				return
			}

			c.SSEvent("log", event)
			flusher.Flush()
		}
	}
}

func ok(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, apiResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func accepted(c *gin.Context, message string, data any) {
	c.JSON(http.StatusAccepted, apiResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func badRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, apiResponse{
		Success: false,
		Message: message,
	})
}

func notFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, apiResponse{
		Success: false,
		Message: message,
	})
}

func internalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, apiResponse{
		Success: false,
		Message: message,
	})
}
