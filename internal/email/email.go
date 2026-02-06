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

type MailSender interface {
	SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

type realMailSender struct{}

func (r *realMailSender) SendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	return smtp.SendMail(addr, a, from, to, msg)
}

type Service struct {
	auth   smtp.Auth
	config SMTPConfig
	sender MailSender
}

func NewService(config SMTPConfig) *Service {
	return NewServiceWithSender(config, &realMailSender{})
}

func NewServiceWithSender(config SMTPConfig, sender MailSender) *Service {
	auth := smtp.PlainAuth("",
		config.Username,
		config.Password,
		config.Host,
	)
	return &Service{
		auth:   auth,
		config: config,
		sender: sender,
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

	if err := s.sender.SendMail(
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
