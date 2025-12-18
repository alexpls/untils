package email

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSend(t *testing.T) {
	s := NewService()
	err := s.Send(context.Background(), &SendParams{
		Recipient: "alexpls@fastmail.com",
		Subject: "A humble test",
		Body: "it works?",
	})
	require.NoError(t, err)
}
