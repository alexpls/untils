package email

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/mail"
	"net/smtp"
	"net/textproto"
)

type SMTPConfig struct {
	Username string
	Password string
	Host     string
	Port     int
	From     string
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
	if config.From == "" {
		panic("smtp from address is required")
	}

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
	HTMLBody  string
}

func (s *Service) Send(ctx context.Context, params *SendParams) error {
	msg, err := buildMessage(s.config.From, params)
	if err != nil {
		return err
	}

	var auth smtp.Auth = nil
	if s.auth != nil && s.config.Username != "" {
		auth = s.auth
	}

	if err := s.sender.SendMail(
		fmt.Sprintf("%s:%d", s.config.Host, s.config.Port),
		auth,
		s.config.From,
		[]string{params.Recipient},
		msg,
	); err != nil {
		return err
	}
	return nil
}

func buildMessage(from string, params *SendParams) ([]byte, error) {
	fromHeader := (&mail.Address{Name: "untils", Address: from}).String()

	if params.HTMLBody == "" {
		return []byte("From: " + fromHeader + "\r\n" +
			"To: " + params.Recipient + "\r\n" +
			"Subject: " + params.Subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			params.Body + "\r\n"), nil
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writeMultipartPart(writer, "text/plain; charset=UTF-8", params.Body); err != nil {
		return nil, fmt.Errorf("writing text email part: %w", err)
	}
	if err := writeMultipartPart(writer, "text/html; charset=UTF-8", params.HTMLBody); err != nil {
		return nil, fmt.Errorf("writing html email part: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart email body: %w", err)
	}

	msg := []byte("From: " + fromHeader + "\r\n" +
		"To: " + params.Recipient + "\r\n" +
		"Subject: " + params.Subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=" + writer.Boundary() + "\r\n" +
		"\r\n" +
		body.String())

	return msg, nil
}

func writeMultipartPart(writer *multipart.Writer, contentType string, body string) error {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType)

	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	if _, err := part.Write([]byte(body)); err != nil {
		return err
	}
	_, err = part.Write([]byte("\r\n"))
	return err
}
