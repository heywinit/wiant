package auth

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"

	"github.com/heywinit/prowl/server/config"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, text string) error
}

type SMTPMailer struct{ cfg config.Config }

func NewSMTPMailer(cfg config.Config) *SMTPMailer { return &SMTPMailer{cfg: cfg} }

func (m *SMTPMailer) Send(ctx context.Context, to, subject, text string) error {
	from, err := mail.ParseAddress(m.cfg.SMTPFrom)
	if err != nil {
		return fmt.Errorf("parse SMTP_FROM: %w", err)
	}

	address := net.JoinHostPort(m.cfg.SMTPHost, fmt.Sprint(m.cfg.SMTPPort))

	var auth smtp.Auth
	if m.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", m.cfg.SMTPUser, m.cfg.SMTPPassword, m.cfg.SMTPHost)
	}
	message := formatMessage(m.cfg.SMTPFrom, to, subject, text)

	result := make(chan error, 1)

	go func() { result <- smtp.SendMail(address, auth, from.Address, []string{to}, message) }()
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func formatMessage(from, to, subject, text string) []byte {
	messageText := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		from, to, subject, text,
	)
	return []byte(messageText)
}
