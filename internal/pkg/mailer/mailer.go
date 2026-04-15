package mailer

import (
	"fmt"
	"log/slog"
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
	subject := "Your CashLens Verification Code"
	htmlBody := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				.otp-code {
					font-size: 32px;
					font-weight: bold;
					color: #4CAF50;
					letter-spacing: 5px;
					background-color: #f4f4f4;
					padding: 15px;
					border-radius: 8px;
					display: inline-block;
					margin: 20px 0;
				}
			</style>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 10px;">
				<h2 style="color: #4CAF50;">Welcome to CashLens!</h2>
				<p>Hello,</p>
				<p>Thank you for joining CashLens. To complete your registration, please use the following 6-digit verification code:</p>
				<div style="text-align: center;">
					<div class="otp-code">%s</div>
				</div>
				<p>This code will expire in 10 minutes.</p>
				<hr style="border: 0; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="font-size: 12px; color: #888;">If you did not create an account, you can safely ignore this email.</p>
			</div>
		</body>
		</html>
	`, token)

	// In development, we might want to just log it if SMTP is not configured
	if m.config.User == "" || m.config.Password == "" {
		slog.Info("DEBUG: HTML Email Sent", "to", to, "otp", token)
		return nil
	}

	// Always log OTP to terminal for development ease
	slog.Info("📧 [MAILER] Sending OTP", "to", to, "otp", token)

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
