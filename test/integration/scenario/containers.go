package scenario

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	GotenbergDockerRepository  string
	GotenbergVersion           string
	GotenbergContainerPlatform string
)

type noopLogger struct{}

func (n *noopLogger) Printf(format string, v ...interface{}) {
	// NOOP
}

func startGotenbergContainer(ctx context.Context, env map[string]string) (*testcontainers.DockerNetwork, testcontainers.Container, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	n, err := network.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("create Gotenberg container network: %w", err)
	}

	healthPath := "/health"
	if env["API_ROOT_PATH"] != "" {
		healthPath = fmt.Sprintf("%shealth", env["API_ROOT_PATH"])
	}

	req := testcontainers.ContainerRequest{
		Image:         fmt.Sprintf("gotenberg/%s:%s", GotenbergDockerRepository, GotenbergVersion),
		ImagePlatform: GotenbergContainerPlatform,
		ExposedPorts:  []string{"3000/tcp"},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.ExtraHosts = []string{"host.docker.internal:host-gateway"}
		},
		Networks:   []string{n.Name},
		WaitingFor: wait.ForHTTP(healthPath),
		Env:        env,
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Logger:           &noopLogger{},
	})
	if err != nil {
		err = fmt.Errorf("start new Gotenberg container: %w", err)
	}

	return n, c, err
}

func execCommandInIntegrationToolsContainer(ctx context.Context, cmd []string, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image: "gotenberg/integration-tools:latest",
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      path,
				ContainerFilePath: filepath.Base(path),
				FileMode:          0o700,
			},
		},
		Cmd: []string{"tail", "-f", "/dev/null"}, // Keeps container running indefinitely.
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Logger:           &noopLogger{},
	})
	if err != nil {
		return "", fmt.Errorf("start new Integration Tools container: %w", err)
	}
	defer func(c testcontainers.Container, ctx context.Context) {
		err := c.Terminate(ctx)
		if err != nil {
			fmt.Printf("terminate container: %v\n", err)
		}
	}(c, ctx)

	_, output, err := c.Exec(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("exec %q: %w", cmd, err)
	}

	b, err := io.ReadAll(output)
	if err != nil {
		return "", fmt.Errorf("read output: %w", err)
	}

	return string(b), nil
}

func containerHttpEndpoint(ctx context.Context, container testcontainers.Container, port nat.Port) (string, error) {
	ip, err := container.Host(ctx)
	if err != nil {
		return "", fmt.Errorf("get container IP: %w", err)
	}
	mapped, err := container.MappedPort(ctx, port)
	if err != nil {
		return "", fmt.Errorf("get container port: %w", err)
	}
	return fmt.Sprintf("http://%s:%s", ip, mapped.Port()), nil
}

func containerLogEntries(ctx context.Context, container testcontainers.Container) (string, error) {
	logReader, err := container.Logs(ctx)
	if err != nil {
		return "", fmt.Errorf("get container log entries: %w", err)
	}
	defer logReader.Close()

	logsBytes, err := io.ReadAll(logReader)
	if err != nil {
		return "", fmt.Errorf("read container log entries: %w", err)
	}

	return string(logsBytes), nil
}
