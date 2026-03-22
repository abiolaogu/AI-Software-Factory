package harness

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestEnvOrDefault(t *testing.T) {
	const key = "NF_TEST_SAMPLE_ENV"
	t.Setenv(key, "")
	require.Equal(t, "fallback", envOrDefault(key, "fallback"))

	t.Setenv(key, "value")
	require.Equal(t, "value", envOrDefault(key, "fallback"))
}

type fakeTerminator struct {
	err   error
	calls int
}

func (f *fakeTerminator) Terminate(context.Context, ...testcontainers.TerminateOption) error {
	f.calls++
	return f.err
}

type fakeEndpointContainer struct {
	host    string
	hostErr error
	port    nat.Port
	portErr error
}

func (f *fakeEndpointContainer) Host(context.Context) (string, error) {
	return f.host, f.hostErr
}

func (f *fakeEndpointContainer) MappedPort(context.Context, nat.Port) (nat.Port, error) {
	return f.port, f.portErr
}

func TestTerminateAllStopsOnError(t *testing.T) {
	first := &fakeTerminator{}
	second := &fakeTerminator{err: errors.New("stop")}
	third := &fakeTerminator{}

	err := terminateAll(context.Background(), first, nil, second, third)
	require.EqualError(t, err, "stop")
	require.Equal(t, 1, first.calls)
	require.Equal(t, 1, second.calls)
	require.Equal(t, 0, third.calls)
}

func TestContainerEndpointFormatsAddresses(t *testing.T) {
	address, err := containerEndpoint(context.Background(), &fakeEndpointContainer{
		host: "127.0.0.1",
		port: nat.Port("5433/tcp"),
	}, "5433/tcp", "postgres://yugabyte@%s:%s/yugabyte?sslmode=disable")
	require.NoError(t, err)
	require.Equal(t, "postgres://yugabyte@127.0.0.1:5433/yugabyte?sslmode=disable", address)

	_, err = containerEndpoint(context.Background(), &fakeEndpointContainer{hostErr: errors.New("host failed")}, "5433/tcp", "%s:%s")
	require.EqualError(t, err, "host failed")

	_, err = containerEndpoint(context.Background(), &fakeEndpointContainer{
		host:    "127.0.0.1",
		portErr: errors.New("port failed"),
	}, "5433/tcp", "%s:%s")
	require.EqualError(t, err, "port failed")
}
