package main

import (
	"fmt"
	"log"

	"github.com/CodeEnthusiast09/mini-brimble/server/internal/api"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/caddy"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/config"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/database"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/deployment"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/deploymentstore"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/docker"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/logstore"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/logstream"
	"github.com/CodeEnthusiast09/mini-brimble/server/internal/router"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[env] no .env file found, using system environment")
	}

	cfg := config.Load()

	db, err := database.Connect(cfg.DBConfig)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	dockerClient, err := docker.NewClient(cfg.DockerConfig.SocketPath)
	if err != nil {
		log.Fatalf("create docker client: %v", err)
	}

	deployments := deploymentstore.NewStore(db)
	logs := logstore.NewStore(db)
	streams := logstream.NewHub()

	caddyClient := caddy.New(fmt.Sprintf("http://%s:%d", cfg.CaddyConfig.Host, cfg.CaddyConfig.Port))
	deploymentService := deployment.NewService(
		deployments,
		logs,
		streams,
		dockerClient,
		caddyClient,
		"",
		cfg.AppBaseURL,
		cfg.DeploymentUpstreamHost,
	)

	deploymentHandler := api.NewDeploymentHandler(deploymentService, deployments, logs, streams)

	r := gin.New()
	router.Setup(r, router.Handlers{
		Deployment: deploymentHandler,
	}, db)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("mini-brimble running on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
