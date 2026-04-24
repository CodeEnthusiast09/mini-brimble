package docker

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	dockerclient "github.com/moby/moby/client"
)

type Client struct {
	socketPath string
	cli        *dockerclient.Client
}

func NewClient(socketPath string) (*Client, error) {
	cli, err := dockerclient.New(dockerclient.WithHost("unix://" + socketPath))
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	return &Client{
		socketPath: socketPath,
		cli:        cli,
	}, nil
}

func (c *Client) GetFreePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("listen on random port: %w", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func (c *Client) RunContainer(ctx context.Context, image string, hostPort int) (string, error) {
	containerPort, err := network.ParsePort("8080/tcp")
	if err != nil {
		return "", fmt.Errorf("parse container port: %w", err)
	}

	resp, err := c.cli.ContainerCreate(ctx, dockerclient.ContainerCreateOptions{
		Config: &container.Config{Image: image},
		HostConfig: &container.HostConfig{
			PortBindings: network.PortMap{
				containerPort: []network.PortBinding{
					{
						HostIP:   netip.MustParseAddr("0.0.0.0"),
						HostPort: fmt.Sprintf("%d", hostPort),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	_, err = c.cli.ContainerStart(ctx, resp.ID, dockerclient.ContainerStartOptions{})
	if err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}

	return resp.ID, nil
}

func (c *Client) StreamLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	logs, err := c.cli.ContainerLogs(ctx, containerID, dockerclient.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	})
	if err != nil {
		return nil, fmt.Errorf("stream logs: %w", err)
	}

	return logs, nil
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10
	_, err := c.cli.ContainerStop(ctx, containerID, dockerclient.ContainerStopOptions{Timeout: &timeout})
	if err != nil {
		return fmt.Errorf("stop container: %w", err)
	}

	return nil
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	_, err := c.cli.ContainerRemove(ctx, containerID, dockerclient.ContainerRemoveOptions{Force: true})
	if err != nil {
		return fmt.Errorf("remove container: %w", err)
	}

	return nil
}
