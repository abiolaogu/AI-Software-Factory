package harness

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type endpointContainer interface {
	Host(context.Context) (string, error)
	MappedPort(context.Context, nat.Port) (nat.Port, error)
}

type terminator interface {
	Terminate(context.Context, ...testcontainers.TerminateOption) error
}

type Stack struct {
	Yugabyte      testcontainers.Container
	Dragonfly     testcontainers.Container
	NATS          testcontainers.Container
	Qdrant        testcontainers.Container
	YugabyteDSN   string
	DragonflyAddr string
	NATSURL       string
	QdrantURL     string
}

func Start(ctx context.Context) (*Stack, error) {
	stack := &Stack{}
	containers := make([]testcontainers.Container, 0, 4)

	cleanup := func() {
		terminateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = terminateAll(terminateCtx, toTerminators(containers)...)
	}

	yugabyte, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        envOrDefault("NF_TEST_YUGABYTE_IMAGE", "yugabytedb/yugabyte:2025.2.1.0-b141"),
			Cmd:          []string{"bin/yugabyted", "start", "--background=false"},
			ExposedPorts: []string{"5433/tcp"},
			WaitingFor:   wait.ForListeningPort("5433/tcp").WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		cleanup()
		return nil, err
	}
	containers = append(containers, yugabyte)

	dragonfly, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        envOrDefault("NF_TEST_DRAGONFLY_IMAGE", "docker.dragonflydb.io/dragonflydb/dragonfly:v1.32.0"),
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		cleanup()
		return nil, err
	}
	containers = append(containers, dragonfly)

	natsContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        envOrDefault("NF_TEST_NATS_IMAGE", "nats:2.11-alpine"),
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForListeningPort("4222/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		cleanup()
		return nil, err
	}
	containers = append(containers, natsContainer)

	qdrant, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        envOrDefault("NF_TEST_QDRANT_IMAGE", "qdrant/qdrant:v1.13.6"),
			ExposedPorts: []string{"6333/tcp"},
			WaitingFor:   wait.ForListeningPort("6333/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		cleanup()
		return nil, err
	}
	containers = append(containers, qdrant)

	stack.Yugabyte = yugabyte
	stack.Dragonfly = dragonfly
	stack.NATS = natsContainer
	stack.Qdrant = qdrant

	stack.YugabyteDSN, err = containerEndpoint(ctx, yugabyte, "5433/tcp", "postgres://yugabyte@%s:%s/yugabyte?sslmode=disable")
	if err != nil {
		cleanup()
		return nil, err
	}

	stack.DragonflyAddr, err = containerEndpoint(ctx, dragonfly, "6379/tcp", "%s:%s")
	if err != nil {
		cleanup()
		return nil, err
	}

	stack.NATSURL, err = containerEndpoint(ctx, natsContainer, "4222/tcp", "nats://%s:%s")
	if err != nil {
		cleanup()
		return nil, err
	}

	stack.QdrantURL, err = containerEndpoint(ctx, qdrant, "6333/tcp", "http://%s:%s")
	if err != nil {
		cleanup()
		return nil, err
	}

	return stack, nil
}

func (s *Stack) Terminate(ctx context.Context) error {
	return terminateAll(ctx, s.Qdrant, s.NATS, s.Dragonfly, s.Yugabyte)
}

func envOrDefault(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}

func terminateAll(ctx context.Context, containers ...terminator) error {
	for _, container := range containers {
		if container == nil {
			continue
		}
		if err := container.Terminate(ctx); err != nil {
			return err
		}
	}

	return nil
}

func containerEndpoint(ctx context.Context, container endpointContainer, port nat.Port, format string) (string, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return "", err
	}

	mappedPort, err := container.MappedPort(ctx, port)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(format, host, mappedPort.Port()), nil
}

func toTerminators(containers []testcontainers.Container) []terminator {
	result := make([]terminator, 0, len(containers))
	for _, container := range containers {
		result = append(result, container)
	}
	return result
}
