package messaging_test

import (
	"context"
	"testing"

	"github.com/stratogroup/neuroforge-sdx/internal/adapters/messaging"
	"github.com/stretchr/testify/require"
)

func TestNewWithoutURLReturnsNoopPublisher(t *testing.T) {
	publisher, err := messaging.New("", "subject")
	require.NoError(t, err)
	require.NoError(t, publisher.Publish(context.Background(), "subject", []byte("payload")))
	require.NoError(t, publisher.Close())
}

func TestNoopPublisherDoesNothing(t *testing.T) {
	publisher := messaging.NoopPublisher{}
	require.NoError(t, publisher.Publish(context.Background(), "subject", []byte("payload")))
	require.NoError(t, publisher.Close())
}
