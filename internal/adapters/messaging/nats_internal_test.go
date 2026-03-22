package messaging

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestPublisherPublishesToDefaultSubject(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = listener.Close()
	}()

	received := make(chan string, 1)
	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		if _, err := conn.Write([]byte("INFO {\"max_payload\":1048576,\"headers\":true}\r\n")); err != nil {
			serverDone <- err
			return
		}

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					serverDone <- nil
					return
				}
				serverDone <- err
				return
			}

			switch {
			case strings.HasPrefix(line, "CONNECT"):
			case strings.HasPrefix(line, "PING"):
				if _, err := conn.Write([]byte("PONG\r\n")); err != nil {
					serverDone <- err
					return
				}
			case strings.HasPrefix(line, "PUB "):
				fields := strings.Fields(strings.TrimSpace(line))
				size, err := strconv.Atoi(fields[2])
				if err != nil {
					serverDone <- err
					return
				}

				payload := make([]byte, size+2)
				if _, err := io.ReadFull(reader, payload); err != nil {
					serverDone <- err
					return
				}

				received <- fmt.Sprintf("%s|%s", fields[1], string(payload[:size]))
				serverDone <- nil
				return
			case strings.HasPrefix(line, "HPUB "):
				fields := strings.Fields(strings.TrimSpace(line))
				headerSize, err := strconv.Atoi(fields[2])
				if err != nil {
					serverDone <- err
					return
				}

				totalSize, err := strconv.Atoi(fields[3])
				if err != nil {
					serverDone <- err
					return
				}

				payload := make([]byte, totalSize+2)
				if _, err := io.ReadFull(reader, payload); err != nil {
					serverDone <- err
					return
				}

				body := string(payload[headerSize : totalSize])
				received <- fmt.Sprintf("%s|%s", fields[1], body)
				serverDone <- nil
				return
			}
		}
	}()

	publisher, err := New("nats://"+listener.Addr().String(), "events.workflow")
	require.NoError(t, err)
	require.NoError(t, publisher.Publish(context.Background(), "", []byte("payload")))
	require.NoError(t, publisher.Close())

	select {
	case message := <-received:
		require.Equal(t, "events.workflow|payload", message)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	require.NoError(t, <-serverDone)
}

func TestPublisherUsesJetStreamAndCarriesRequestMetadata(t *testing.T) {
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer otel.SetTextMapPropagator(previousPropagator)

	ctx := domain.WithTenantID(context.Background(), "tenant-a")
	ctx = domain.WithIdempotencyKey(ctx, "idem-1")
	tracerProvider := sdktrace.NewTracerProvider()
	defer func() {
		_ = tracerProvider.Shutdown(context.Background())
	}()
	tracer := tracerProvider.Tracer("messaging-test")
	var span trace.Span
	ctx, span = tracer.Start(ctx, "publish")
	defer span.End()

	js := &stubJetStreamManager{}
	conn := &stubConn{}
	publisher := &Publisher{
		conn:      conn,
		jetstream: js,
		subject:   "events.workflow",
		stream:    "regional-events",
		now: func() time.Time {
			return time.Date(2026, time.March, 22, 12, 0, 0, 0, time.UTC)
		},
	}

	err := publisher.Publish(ctx, "", []byte(`{"status":"ok"}`))
	require.NoError(t, err)
	require.Nil(t, conn.message)
	require.NotNil(t, js.message)
	require.Equal(t, "events.workflow", js.message.Subject)
	require.Equal(t, "tenant-a", js.message.Header.Get(domain.HeaderTenantID))
	require.Equal(t, "idem-1", js.message.Header.Get(domain.HeaderIdempotencyKey))
	require.Equal(t, "regional-events", js.message.Header.Get("Nats-Expected-Stream"))
	require.Equal(t, "application/json", js.message.Header.Get("Content-Type"))
	require.Equal(t, "2026-03-22T12:00:00Z", js.message.Header.Get("Nf-Event-Published-At"))
	require.NotEmpty(t, js.message.Header.Get("traceparent"))
}

func TestPublisherFallsBackToCoreNATSWhenJetStreamIsDisabled(t *testing.T) {
	conn := &stubConn{}
	publisher := &Publisher{
		conn:    conn,
		subject: "events.workflow",
		now: func() time.Time {
			return time.Date(2026, time.March, 22, 12, 0, 0, 0, time.UTC)
		},
	}

	err := publisher.Publish(context.Background(), "workflow.created", []byte("payload"))
	require.NoError(t, err)
	require.NotNil(t, conn.message)
	require.Equal(t, "workflow.created", conn.message.Subject)
	require.Equal(t, "application/json", conn.message.Header.Get("Content-Type"))
}

func TestPublisherCloseDrainsConnection(t *testing.T) {
	conn := &stubConn{}
	publisher := &Publisher{conn: conn}

	require.NoError(t, publisher.Close())
	require.True(t, conn.drained)
}

func TestNewEnsuresJetStreamStreamWhenConfigured(t *testing.T) {
	previousConnect := connectNATS
	defer func() {
		connectNATS = previousConnect
	}()

	js := &stubJetStreamManager{streamInfoErr: nats.ErrStreamNotFound}
	connectNATS = func(string, ...nats.Option) (natsJetStreamConnection, error) {
		return &stubJetStreamConn{jetstream: js}, nil
	}

	publisher, err := New("nats://127.0.0.1:4222", "events.workflow", WithJetStreamStream("regional-events"))
	require.NoError(t, err)
	defer publisher.Close() //nolint:errcheck

	require.NotNil(t, js.addedConfig)
	require.Equal(t, "regional-events", js.addedConfig.Name)
	require.Equal(t, []string{"events.workflow"}, js.addedConfig.Subjects)
}

func TestNewUpdatesJetStreamStreamWhenSubjectIsMissing(t *testing.T) {
	previousConnect := connectNATS
	defer func() {
		connectNATS = previousConnect
	}()

	js := &stubJetStreamManager{
		streamInfo: &nats.StreamInfo{
			Config: nats.StreamConfig{
				Name:     "regional-events",
				Subjects: []string{"events.existing"},
			},
		},
	}
	connectNATS = func(string, ...nats.Option) (natsJetStreamConnection, error) {
		return &stubJetStreamConn{jetstream: js}, nil
	}

	publisher, err := New("nats://127.0.0.1:4222", "events.workflow", WithJetStreamStream("regional-events"))
	require.NoError(t, err)
	defer publisher.Close() //nolint:errcheck

	require.NotNil(t, js.updatedConfig)
	require.ElementsMatch(t, []string{"events.existing", "events.workflow"}, js.updatedConfig.Subjects)
}

func TestNewReturnsErrorWhenJetStreamProvisioningFails(t *testing.T) {
	previousConnect := connectNATS
	defer func() {
		connectNATS = previousConnect
	}()

	js := &stubJetStreamManager{streamInfoErr: errors.New("jetstream down")}
	connectNATS = func(string, ...nats.Option) (natsJetStreamConnection, error) {
		return &stubJetStreamConn{jetstream: js}, nil
	}

	_, err := New("nats://127.0.0.1:4222", "events.workflow", WithJetStreamStream("regional-events"))
	require.Error(t, err)
	require.ErrorContains(t, err, "jetstream down")
}

type stubConn struct {
	message  *nats.Msg
	drained  bool
	drainErr error
}

func (s *stubConn) PublishMsg(message *nats.Msg) error {
	s.message = message
	return nil
}

func (s *stubConn) Drain() error {
	s.drained = true
	return s.drainErr
}

func (s *stubConn) Close() {}

type stubJetStreamPublisher struct {
	message *nats.Msg
}

func (s *stubJetStreamPublisher) PublishMsg(message *nats.Msg, _ ...nats.PubOpt) (*nats.PubAck, error) {
	s.message = message
	return &nats.PubAck{}, nil
}

type stubJetStreamConn struct {
	stubConn
	jetstream jetStreamContext
	jsErr     error
}

func (s *stubJetStreamConn) JetStream(...nats.JSOpt) (jetStreamContext, error) {
	return s.jetstream, s.jsErr
}

type stubJetStreamManager struct {
	stubJetStreamPublisher
	streamInfo   *nats.StreamInfo
	streamInfoErr error
	addedConfig  *nats.StreamConfig
	updatedConfig *nats.StreamConfig
}

func (s *stubJetStreamManager) StreamInfo(string, ...nats.JSOpt) (*nats.StreamInfo, error) {
	return s.streamInfo, s.streamInfoErr
}

func (s *stubJetStreamManager) AddStream(config *nats.StreamConfig, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	cloned := *config
	s.addedConfig = &cloned
	return &nats.StreamInfo{Config: cloned}, nil
}

func (s *stubJetStreamManager) UpdateStream(config *nats.StreamConfig, _ ...nats.JSOpt) (*nats.StreamInfo, error) {
	cloned := *config
	s.updatedConfig = &cloned
	return &nats.StreamInfo{Config: cloned}, nil
}
