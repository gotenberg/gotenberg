package scenario

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/moby/moby/api/types/container"
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

func (n *noopLogger) Printf(format string, v ...any) {
	// NOOP
}

// integrationAllowList is the default allow-list pattern injected into
// every Gotenberg container started by the integration tests. The outbound
// URL guard introduced for SSRF protection rejects URLs whose host
// resolves to a non-public IP, which would block:
//
//   - host.docker.internal (Docker host gateway, RFC1918)
//   - The static helper server running inside the test network
//   - file:// URIs created in /tmp by the API context
//
// Setting the allow-list to a permissive pattern flips the URL guard into
// "allow-list match bypasses the IP check" mode for every URL the tests
// touch. Operator-supplied deny-lists still apply, so deny-list scenarios
// keep working. Test scenarios that exercise allow-list semantics
// explicitly override this default in their environment table.
//
// Production operators wanting a similar bypass for trusted internal
// destinations should set their own --*-allow-list with a tighter regex
// (for example ^https?://internal\.svc(:|/|$)).
const integrationAllowList = `.+`

// applyDefaultEnv merges baseline environment variables that the
// integration tests rely on into env, without overwriting values supplied
// by the test scenario itself. Tests can clear a default by setting it to
// the empty string in their scenario table.
func applyDefaultEnv(env map[string]string) map[string]string {
	if env == nil {
		env = make(map[string]string)
	}
	defaults := map[string]string{
		"CHROMIUM_ALLOW_LIST":          integrationAllowList,
		"API_DOWNLOAD_FROM_ALLOW_LIST": integrationAllowList,
		"WEBHOOK_ALLOW_LIST":           integrationAllowList,
	}
	for k, v := range defaults {
		if _, ok := env[k]; !ok {
			env[k] = v
		}
	}
	return env
}

func startGotenbergContainer(ctx context.Context, env map[string]string) (*testcontainers.DockerNetwork, testcontainers.Container, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	env = applyDefaultEnv(env)

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
		Image:         "gotenberg/integration-tools:latest",
		ImagePlatform: GotenbergContainerPlatform,
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

func containerHttpEndpoint(ctx context.Context, container testcontainers.Container, port string) (string, error) {
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
