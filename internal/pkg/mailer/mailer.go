package mailer

import (
	"fmt"
	"net/smtp"

	"github.com/rifqimalik/cashlens-backend/internal/config"
)

type Mailer interface {
	SendConfirmationEmail(to, token string) error
}

type smtpMailer struct {
	config config.MailConfig
}

func NewMailer(cfg config.MailConfig) Mailer {
	return &smtpMailer{config: cfg}
}

func (m *smtpMailer) SendConfirmationEmail(to, token string) error {
	confirmURL := fmt.Sprintf("%s/api/v1/auth/confirm?token=%s", m.config.BaseURL, token)
	
	subject := "Confirm your CashLens account"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				.button {
					background-color: #4CAF50;
					border: none;
					color: white;
					padding: 15px 32px;
					text-align: center;
					text-decoration: none;
					display: inline-block;
					font-size: 16px;
					margin: 4px 2px;
					cursor: pointer;
					border-radius: 8px;
				}
			</style>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 10px;">
				<h2 style="color: #4CAF50;">Welcome to CashLens!</h2>
				<p>Hello,</p>
				<p>Thank you for joining CashLens. To start tracking your expenses, please confirm your email address by clicking the button below:</p>
				<div style="text-align: center; margin: 30px 0;">
					<a href="%s" class="button" style="color: white;">Confirm Email Address</a>
				</div>
				<p>Or copy and paste this link into your browser:</p>
				<p style="word-break: break-all; color: #888;">%s</p>
				<p>This link will expire in 24 hours.</p>
				<hr style="border: 0; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="font-size: 12px; color: #888;">If you did not create an account, you can safely ignore this email.</p>
			</div>
		</body>
		</html>
	`, confirmURL, confirmURL)

	// In development, we might want to just log it if SMTP is not configured
	if m.config.User == "" || m.config.Password == "" {
		fmt.Printf("DEBUG: HTML Email to %s Sent\n", to)
		return nil
	}

	header := make(map[string]string)
	header["From"] = m.config.From
	header["To"] = to
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=\"utf-8\""

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + htmlBody

	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	auth := smtp.PlainAuth("", m.config.User, m.config.Password, m.config.Host)

	err := smtp.SendMail(addr, auth, m.config.From, []string{to}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
