package email

import (
	"fmt"

	"github.com/mkobaly/jiraworklog"
	"github.com/wneessen/go-mail"
)

type EmailClient interface {
	SendEmail(subject string, to string, code string) error
}

type FakeEmailClient struct {
}

func (e FakeEmailClient) SendEmail(subject string, to string, code string) error {
	return nil
}

type SmtpClient struct {
	smtpHost string
	port     int
	username string
	password string
}

func NewSmtpClient(cfg *jiraworklog.Config) EmailClient {
	return &SmtpClient{
		smtpHost: cfg.Smtp.Host,
		port:     cfg.Smtp.Port,
		username: cfg.Smtp.UserName,
		password: cfg.Smtp.Password,
	}
}

func (c *SmtpClient) SendEmail(subject string, to string, code string) error {
	message := mail.NewMsg()
	if err := message.From(c.username); err != nil {
		return fmt.Errorf("failed to set From address: %w", err)
	}
	if err := message.To(to); err != nil {
		return fmt.Errorf("failed to set To address: %w", err)
	}

	message.Subject(subject)
	message.SetBodyString(mail.TypeTextPlain, fmt.Sprintf("Your login code is lsted below.\n\n%s", code))

	client, err := mail.NewClient(c.smtpHost, mail.WithPort(c.port), mail.WithTLSPolicy(mail.TLSMandatory),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover), mail.WithUsername(c.username), mail.WithPassword(c.password))
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}
	if err := client.DialAndSend(message); err != nil {
		return fmt.Errorf("failed to send mail: %w", err)
	}
	return nil
}
