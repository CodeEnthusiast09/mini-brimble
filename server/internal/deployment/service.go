package deployment

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/CodeEnthusiast09/mini-brimble/server/internal/caddy"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/deploymentstore"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/docker"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/logstore"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/logstream"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/models"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/railpack"
)

type Service struct {
	deployments      *deploymentstore.Store
	logs             *logstore.Store
	streams          *logstream.Hub
	docker           *docker.Client
	caddy            *caddy.Caddy
	workspaceRoot    string
	publicBaseDomain string
	upstreamHost     string
}

func NewService(
	deployments *deploymentstore.Store,
	logs *logstore.Store,
	streams *logstream.Hub,
	dockerClient *docker.Client,
	caddyClient *caddy.Caddy,
	workspaceRoot string,
	publicBaseDomain string,
	upstreamHost string,
) *Service {
	if workspaceRoot == "" {
		workspaceRoot = os.TempDir()
	}
	if publicBaseDomain == "" {
		publicBaseDomain = "localhost"
	}
	if upstreamHost == "" {
		upstreamHost = "127.0.0.1"
	}

	return &Service{
		deployments:      deployments,
		logs:             logs,
		streams:          streams,
		docker:           dockerClient,
		caddy:            caddyClient,
		workspaceRoot:    workspaceRoot,
		publicBaseDomain: strings.TrimSpace(publicBaseDomain),
		upstreamHost:     strings.TrimSpace(upstreamHost),
	}
}

func (s *Service) Deploy(ctx context.Context, githubURL string) (*models.Deployment, error) {
	deployment := &models.Deployment{
		GithubURL: githubURL,
		Status:    models.StatusPending,
	}
	createErr := s.deployments.Create(ctx, deployment)
	if createErr != nil {
		return nil, fmt.Errorf("create deployment record: %w", createErr)
	}

	s.emitLog(ctx, deployment.ID, "deployment created")

	asyncCtx := context.WithoutCancel(ctx)
	deploymentCopy := *deployment
	go s.runDeploy(asyncCtx, &deploymentCopy)

	return deployment, nil
}

func (s *Service) runDeploy(ctx context.Context, deployment *models.Deployment) {
	logWriter := newLogWriter(ctx, s, deployment.ID)

	workDir, workspaceErr := os.MkdirTemp(s.workspaceRoot, "deployment-"+deployment.ID+"-")
	if workspaceErr != nil {
		s.failDeployment(ctx, deployment, fmt.Sprintf("create workspace: %v", workspaceErr))
		return
	}
	defer os.RemoveAll(workDir)

	projectDir := filepath.Join(workDir, "repo")
	routeHost := s.routeHost(deployment.ID)
	imageName := s.imageName(deployment.ID)

	s.emitLog(ctx, deployment.ID, "cloning repository")
	cloneErr := cloneRepo(ctx, deployment.GithubURL, projectDir, logWriter)
	if cloneErr != nil {
		logWriter.Flush()
		s.failDeployment(ctx, deployment, fmt.Sprintf("clone repository: %v", cloneErr))
		return
	}
	logWriter.Flush()

	deployment.Status = models.StatusBuilding
	deployment.ImageTag = imageName
	buildingUpdateErr := s.deployments.Update(ctx, deployment)
	if buildingUpdateErr != nil {
		s.failDeployment(ctx, deployment, fmt.Sprintf("persist building state: %v", buildingUpdateErr))
		return
	}

	s.emitLog(ctx, deployment.ID, "building image with railpack")
	buildErr := railpack.Build(ctx, projectDir, imageName, logWriter)
	if buildErr != nil {
		logWriter.Flush()
		s.failDeployment(ctx, deployment, fmt.Sprintf("build image: %v", buildErr))
		return
	}
	logWriter.Flush()

	hostPort, portErr := s.docker.GetFreePort()
	if portErr != nil {
		s.failDeployment(ctx, deployment, fmt.Sprintf("allocate port: %v", portErr))
		return
	}

	deployment.Status = models.StatusDeploying
	deployment.ContainerPort = hostPort
	deployment.LiveURL = "http://" + routeHost
	deployingUpdateErr := s.deployments.Update(ctx, deployment)
	if deployingUpdateErr != nil {
		s.failDeployment(ctx, deployment, fmt.Sprintf("persist deploying state: %v", deployingUpdateErr))
		return
	}

	s.emitLog(ctx, deployment.ID, fmt.Sprintf("starting container on port %d", hostPort))
	containerID, runErr := s.docker.RunContainer(ctx, imageName, hostPort)
	if runErr != nil {
		s.failDeployment(ctx, deployment, fmt.Sprintf("run container: %v", runErr))
		return
	}
	deployment.ContainerID = containerID
	containerUpdateErr := s.deployments.Update(ctx, deployment)
	if containerUpdateErr != nil {
		s.cleanupContainer(ctx, containerID)
		s.failDeployment(ctx, deployment, fmt.Sprintf("persist container state: %v", containerUpdateErr))
		return
	}

	s.emitLog(ctx, deployment.ID, "adding caddy route")
	routeErr := s.caddy.AddRoute(ctx, routeHost, s.upstreamHost, hostPort)
	if routeErr != nil {
		s.cleanupContainer(ctx, containerID)
		s.failDeployment(ctx, deployment, fmt.Sprintf("add caddy route: %v", routeErr))
		return
	}

	deployment.Status = models.StatusRunning
	runningUpdateErr := s.deployments.Update(ctx, deployment)
	if runningUpdateErr != nil {
		s.cleanupRoute(ctx, routeHost)
		s.cleanupContainer(ctx, containerID)
		s.failDeployment(ctx, deployment, fmt.Sprintf("persist running state: %v", runningUpdateErr))
		return
	}

	s.emitLog(ctx, deployment.ID, "deployment is live at "+deployment.LiveURL)
}

func (s *Service) Stop(ctx context.Context, deploymentID string) error {
	deployment, loadErr := s.deployments.GetByID(ctx, deploymentID)
	if loadErr != nil {
		return fmt.Errorf("load deployment: %w", loadErr)
	}

	s.emitLog(ctx, deployment.ID, "stopping deployment")

	if host := hostFromURL(deployment.LiveURL); host != "" {
		removeRouteErr := s.caddy.RemoveRoute(ctx, host)
		if removeRouteErr != nil {
			return fmt.Errorf("remove caddy route: %w", removeRouteErr)
		}
	}

	if deployment.ContainerID != "" {
		stopErr := s.docker.StopContainer(ctx, deployment.ContainerID)
		if stopErr != nil {
			return fmt.Errorf("stop container: %w", stopErr)
		}
		removeContainerErr := s.docker.RemoveContainer(ctx, deployment.ContainerID)
		if removeContainerErr != nil {
			return fmt.Errorf("remove container: %w", removeContainerErr)
		}
	}

	deleteErr := s.deployments.Delete(ctx, deploymentID)
	if deleteErr != nil {
		return fmt.Errorf("delete deployment record: %w", deleteErr)
	}

	return nil
}

func cloneRepo(ctx context.Context, repoURL string, destDir string, output *logWriter) error {
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, destDir)
	if output != nil {
		cmd.Stdout = output
		cmd.Stderr = output
	}

	return cmd.Run()
}

func (s *Service) emitLog(ctx context.Context, deploymentID string, message string) {
	_ = s.logs.Save(ctx, deploymentID, message)
	s.streams.Broadcast(deploymentID, message)
}

func (s *Service) failDeployment(ctx context.Context, deployment *models.Deployment, message string) {
	s.emitLog(ctx, deployment.ID, message)
	deployment.Status = models.StatusFailed
	_ = s.deployments.Update(ctx, deployment)
}

func (s *Service) cleanupContainer(ctx context.Context, containerID string) {
	if containerID == "" {
		return
	}

	_ = s.docker.StopContainer(ctx, containerID)
	_ = s.docker.RemoveContainer(ctx, containerID)
}

func (s *Service) cleanupRoute(ctx context.Context, host string) {
	if host == "" {
		return
	}

	_ = s.caddy.RemoveRoute(ctx, host)
}

func (s *Service) imageName(deploymentID string) string {
	return "mini-brimble-" + strings.ToLower(deploymentID)
}

func (s *Service) routeHost(deploymentID string) string {
	baseDomain := strings.TrimSpace(s.publicBaseDomain)
	baseDomain = strings.TrimPrefix(baseDomain, "http://")
	baseDomain = strings.TrimPrefix(baseDomain, "https://")
	baseDomain = strings.TrimSuffix(baseDomain, "/")

	if baseDomain == "" {
		baseDomain = "localhost"
	}

	return deploymentID + "." + baseDomain
}

func hostFromURL(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "https://")
	return strings.TrimSuffix(raw, "/")
}

type logWriter struct {
	ctx          context.Context
	service      *Service
	deploymentID string
	pending      string
}

func newLogWriter(ctx context.Context, service *Service, deploymentID string) *logWriter {
	return &logWriter{
		ctx:          ctx,
		service:      service,
		deploymentID: deploymentID,
	}
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.pending += string(p)

	for {
		index := strings.IndexByte(w.pending, '\n')
		if index == -1 {
			break
		}

		line := strings.TrimSpace(w.pending[:index])
		w.pending = w.pending[index+1:]

		if line != "" {
			w.service.emitLog(w.ctx, w.deploymentID, line)
		}
	}

	return len(p), nil
}

func (w *logWriter) Flush() {
	line := strings.TrimSpace(w.pending)
	if line != "" {
		w.service.emitLog(w.ctx, w.deploymentID, line)
	}
	w.pending = ""
}
