package mailer

import (
	"fmt"
	"log/slog"
	"net/smtp"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/config"
)

type Mailer struct {
	settings *config.Settings
}

func New(settings *config.Settings) *Mailer {
	return &Mailer{settings: settings}
}

func (m *Mailer) Send(recipient, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.settings.SMTPHost, m.settings.SMTPPort)
	message := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", recipient, subject, body))

	var auth smtp.Auth
	if m.settings.SMTPUser != "" && m.settings.SMTPPassword != "" {
		auth = smtp.PlainAuth("", m.settings.SMTPUser, m.settings.SMTPPassword, m.settings.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, m.settings.SMTPSender, []string{recipient}, message); err != nil {
		slog.Warn("failed to send email", "recipient", recipient, "error", err)
		return err
	}
	return nil
}
