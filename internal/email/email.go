package email

import (
	"context"
	"fmt"
	"net/smtp"
)

type SMTPConfig struct {
	Username string
	Password string
	Host     string
	Port     int
}

type Service struct {
	auth   smtp.Auth
	config SMTPConfig
}

func NewService(config SMTPConfig) *Service {
	auth := smtp.PlainAuth("",
		config.Username,
		config.Password,
		config.Host,
	)
	return &Service{
		auth:   auth,
		config: config,
	}
}

type SendParams struct {
	Recipient string
	Subject   string
	Body      string
}

func (s *Service) Send(ctx context.Context, params *SendParams) error {
	msg := []byte("To: " + params.Recipient + "\r\n" +
		"Subject: " + params.Subject + "\r\n" +
		"\r\n" +
		params.Body + "\r\n")

	var auth smtp.Auth = nil
	if s.auth != nil && s.config.Username != "" {
		auth = s.auth
	}

	if err := smtp.SendMail(
		fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		auth,
		"notifications@untils.com",
		[]string{params.Recipient},
		msg,
	); err != nil {
		return err
	}
	return nil
}
