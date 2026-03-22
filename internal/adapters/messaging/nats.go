package messaging

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stratogroup/neuroforge-sdx/internal/domain"
	"github.com/stratogroup/neuroforge-sdx/internal/ports"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type natsConnection interface {
	PublishMsg(*nats.Msg) error
	Drain() error
	Close()
}

type natsJetStreamConnection interface {
	natsConnection
	JetStream(...nats.JSOpt) (jetStreamContext, error)
}

type jetStreamPublisher interface {
	PublishMsg(*nats.Msg, ...nats.PubOpt) (*nats.PubAck, error)
}

type jetStreamContext interface {
	jetStreamPublisher
	StreamInfo(string, ...nats.JSOpt) (*nats.StreamInfo, error)
	AddStream(*nats.StreamConfig, ...nats.JSOpt) (*nats.StreamInfo, error)
	UpdateStream(*nats.StreamConfig, ...nats.JSOpt) (*nats.StreamInfo, error)
}

type options struct {
	stream string
}

type Option func(*options)

var connectNATS = func(url string, optionFns ...nats.Option) (natsJetStreamConnection, error) {
	conn, err := nats.Connect(url, optionFns...)
	if err != nil {
		return nil, err
	}

	return &jetStreamConnection{Conn: conn}, nil
}

func WithJetStreamStream(stream string) Option {
	return func(options *options) {
		options.stream = stream
	}
}

type Publisher struct {
	conn      natsConnection
	jetstream jetStreamContext
	subject   string
	stream    string
	now       func() time.Time
}

type NoopPublisher struct{}

type jetStreamConnection struct {
	*nats.Conn
}

func (c *jetStreamConnection) JetStream(optionFns ...nats.JSOpt) (jetStreamContext, error) {
	return c.Conn.JetStream(optionFns...)
}

func New(url, subject string, optionFns ...Option) (ports.EventPublisher, error) {
	if url == "" {
		return NoopPublisher{}, nil
	}

	options := options{}
	for _, optionFn := range optionFns {
		if optionFn != nil {
			optionFn(&options)
		}
	}

	conn, err := connectNATS(url, nats.Timeout(2*time.Second))
	if err != nil {
		return nil, err
	}

	publisher := &Publisher{
		conn:    conn,
		subject: subject,
		stream:  options.stream,
		now:     time.Now,
	}
	if options.stream != "" {
		jetstream, err := conn.JetStream()
		if err != nil {
			conn.Close()
			return nil, err
		}
		if err := ensureStream(jetstream, options.stream, subject); err != nil {
			conn.Close()
			return nil, err
		}
		publisher.jetstream = jetstream
	}

	return publisher, nil
}

func (p *Publisher) Publish(ctx context.Context, subject string, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	message := p.buildMessage(ctx, subject, payload)
	if p.jetstream != nil {
		_, err := p.jetstream.PublishMsg(message)
		return err
	}

	return p.conn.PublishMsg(message)
}

func (p *Publisher) Close() error {
	if p.conn != nil {
		if err := p.conn.Drain(); err != nil {
			p.conn.Close()
			return err
		}
	}

	return nil
}

func (p *Publisher) buildMessage(ctx context.Context, subject string, payload []byte) *nats.Msg {
	if subject == "" {
		subject = p.subject
	}

	message := nats.NewMsg(subject)
	message.Data = payload
	message.Header.Set("Content-Type", "application/json")
	message.Header.Set("Nf-Event-Subject", subject)
	message.Header.Set("Nf-Event-Published-At", p.now().UTC().Format(time.RFC3339))
	if p.stream != "" {
		message.Header.Set("Nats-Expected-Stream", p.stream)
	}
	if tenantID, err := domain.TenantIDFromContext(ctx, "messaging.publish"); err == nil {
		message.Header.Set(domain.HeaderTenantID, tenantID)
	}
	if idempotencyKey := domain.IdempotencyKeyFromContext(ctx); idempotencyKey != "" {
		message.Header.Set(domain.HeaderIdempotencyKey, idempotencyKey)
	}
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		message.Header.Set("traceparent", fmt.Sprintf("00-%s-%s-%02x", spanContext.TraceID(), spanContext.SpanID(), byte(spanContext.TraceFlags())))
	}

	carrier := propagation.HeaderCarrier(message.Header)
	propagation.TraceContext{}.Inject(ctx, carrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return message
}

func ensureStream(jetstream jetStreamContext, streamName, subject string) error {
	info, err := jetstream.StreamInfo(streamName)
	switch {
	case err == nil:
		if slices.Contains(info.Config.Subjects, subject) {
			return nil
		}

		updated := info.Config
		updated.Subjects = append(updated.Subjects, subject)
		_, err = jetstream.UpdateStream(&updated)
		return err
	case errors.Is(err, nats.ErrStreamNotFound):
		_, err = jetstream.AddStream(&nats.StreamConfig{
			Name:      streamName,
			Subjects:  []string{subject},
			Storage:   nats.FileStorage,
			Retention: nats.LimitsPolicy,
			Replicas:  1,
			MaxAge:    7 * 24 * time.Hour,
		})
		return err
	default:
		return err
	}
}

func (NoopPublisher) Publish(context.Context, string, []byte) error {
	return nil
}

func (NoopPublisher) Close() error {
	return nil
}
