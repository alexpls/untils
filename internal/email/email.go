package email

import (
	"context"
	"net/smtp"
)

type Service struct {
	auth smtp.Auth
}

func NewService() *Service {
	// TODO: implement auth...
	auth := smtp.PlainAuth("", "", "", "")
	return &Service{
		auth: auth,
	}
}

type SendParams struct {
	Recipient string
	Subject string
	Body string
}

func (s *Service) Send(ctx context.Context, params *SendParams) error {
	msg := []byte("To: " + params.Recipient + "\r\n" +
		"Subject: " + params.Subject + "\r\n" +
		"\r\n" +
		params.Body + "\r\n")

	if err := smtp.SendMail(
		"127.0.0.1:1025",
		nil, // TODO: implement auth
		"notifications@untils.com",
		[]string{params.Recipient},
		msg,
	); err != nil {
		return err
	}
	return nil
}
