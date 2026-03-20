package email

import (
	"context"
	"errors"
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockMailSender struct {
	calls []sendMailCall
	err   error
}

type sendMailCall struct {
	addr string
	a    smtp.Auth
	from string
	to   []string
	msg  []byte
}

func (m *mockMailSender) SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	m.calls = append(m.calls, sendMailCall{
		addr: addr,
		a:    a,
		from: from,
		to:   to,
		msg:  msg,
	})
	return m.err
}

func TestSend(t *testing.T) {
	t.Run("sends email with correct parameters", func(t *testing.T) {
		mock := &mockMailSender{}
		s := NewServiceWithSender(SMTPConfig{
			Host:     "smtp.example.com",
			Port:     587,
			Username: "user@example.com",
			Password: "secret",
			From:     "alerts@example.com",
		}, mock)

		err := s.Send(context.Background(), &SendParams{
			Recipient: "alex@example.com",
			Subject:   "A humble test",
			Body:      "it works?",
		})

		require.NoError(t, err)
		require.Len(t, mock.calls, 1)
		call := mock.calls[0]
		require.Equal(t, "smtp.example.com:587", call.addr)
		require.Equal(t, "alerts@example.com", call.from)
		require.Equal(t, []string{"alex@example.com"}, call.to)
		require.Contains(t, string(call.msg), "From: \"untils\" <alerts@example.com>")
		require.Contains(t, string(call.msg), "Subject: A humble test")
		require.Contains(t, string(call.msg), "Content-Type: text/plain; charset=UTF-8")
		require.Contains(t, string(call.msg), "it works?")
	})

	t.Run("sends multipart email when html body is present", func(t *testing.T) {
		mock := &mockMailSender{}
		s := NewServiceWithSender(SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "notifications@untils.com",
		}, mock)

		err := s.Send(context.Background(), &SendParams{
			Recipient: "alex@example.com",
			Subject:   "Multipart test",
			Body:      "plain text body",
			HTMLBody:  "<p>html body</p>",
		})

		require.NoError(t, err)
		require.Len(t, mock.calls, 1)

		msg := string(mock.calls[0].msg)
		require.Contains(t, msg, "From: \"untils\" <notifications@untils.com>")
		require.Contains(t, msg, "Subject: Multipart test")
		require.Contains(t, msg, "MIME-Version: 1.0")
		require.Contains(t, msg, "Content-Type: multipart/alternative; boundary=")
		require.Contains(t, msg, "Content-Type: text/plain; charset=UTF-8")
		require.Contains(t, msg, "Content-Type: text/html; charset=UTF-8")
		require.Contains(t, msg, "plain text body")
		require.Contains(t, msg, "<p>html body</p>")
	})

	t.Run("returns error when sender fails", func(t *testing.T) {
		mock := &mockMailSender{err: errors.New("smtp error")}
		s := NewServiceWithSender(SMTPConfig{
			Host: "127.0.0.1",
			Port: 1025,
			From: "notifications@untils.com",
		}, mock)

		err := s.Send(context.Background(), &SendParams{
			Recipient: "test@example.com",
			Subject:   "Test",
			Body:      "Body",
		})

		require.Error(t, err)
		require.Equal(t, "smtp error", err.Error())
	})

	t.Run("works without auth when no credentials", func(t *testing.T) {
		mock := &mockMailSender{}
		s := NewServiceWithSender(SMTPConfig{
			Host: "localhost",
			Port: 1025,
			From: "notifications@untils.com",
		}, mock)

		err := s.Send(context.Background(), &SendParams{
			Recipient: "test@example.com",
			Subject:   "Test",
			Body:      "Body",
		})

		require.NoError(t, err)
		require.Len(t, mock.calls, 1)
		require.Nil(t, mock.calls[0].a)
		require.Equal(t, "notifications@untils.com", mock.calls[0].from)
	})

	t.Run("panics when from address is missing", func(t *testing.T) {
		require.PanicsWithValue(t, "smtp from address is required", func() {
			NewServiceWithSender(SMTPConfig{
				Host: "localhost",
				Port: 1025,
			}, &mockMailSender{})
		})
	})
}
