package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port         int
	DBConfig     DBConfig
	DockerConfig DockerConfig
	CaddyConfig  CaddyConfig
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type DockerConfig struct {
	SocketPath string
}

type CaddyConfig struct {
	Host string
	Port int
}

func Load() *Config {
	return &Config{
		Port: getEnvInt("PORT", 8080),
		DBConfig: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			Name:     getEnv("DB_NAME", "railpack"),
		},
		DockerConfig: DockerConfig{
			SocketPath: getEnv("DOCKER_SOCKET_PATH", "/var/run/docker.sock"),
		},
		CaddyConfig: CaddyConfig{
			Host: getEnv("CADDY_HOST", "localhost"),
			Port: getEnvInt("CADDY_PORT", 2019),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if intValue, err := strconv.Atoi(v); err == nil {
			return intValue
		}
	}

	return defaultValue
}
