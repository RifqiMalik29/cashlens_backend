package mailer

import (
	"fmt"
	"log/slog"
	"net/smtp"

	"github.com/rifqimalik/cashlens-backend/internal/config"
)

type Mailer interface {
	SendConfirmationEmail(to, token string) error
	SendTrialExpiredEmail(to string) error
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

func (m *smtpMailer) SendTrialExpiredEmail(to string) error {
	subject := "Your CashLens Free Trial Has Ended"
	htmlBody := `
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				.cta-button {
					display: inline-block;
					background-color: #4CAF50;
					color: white;
					padding: 12px 24px;
					border-radius: 6px;
					text-decoration: none;
					font-weight: bold;
					margin: 20px 0;
				}
			</style>
		</head>
		<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
			<div style="max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #ddd; border-radius: 10px;">
				<h2 style="color: #4CAF50;">Your Free Trial Has Ended</h2>
				<p>Hello,</p>
				<p>Your 7-day free trial of CashLens Premium has ended. You now have access to the free tier (50 transactions and 5 receipt scans per month).</p>
				<p>Upgrade to Premium to get unlimited transactions, unlimited receipt scans, and full access to all features.</p>
				<div style="text-align: center;">
					<a href="https://cashlens.app" class="cta-button">Upgrade to Premium</a>
				</div>
				<hr style="border: 0; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="font-size: 12px; color: #888;">You received this email because you had a free trial with CashLens.</p>
			</div>
		</body>
		</html>
	`

	// In development, we might want to just log it if SMTP is not configured
	if m.config.User == "" || m.config.Password == "" {
		slog.Info("DEBUG: Trial Expired Email Sent", "to", to)
		return nil
	}

	slog.Info("📧 [MAILER] Sending trial expired email", "to", to)

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

	if err := smtp.SendMail(addr, auth, m.config.From, []string{to}, []byte(message)); err != nil {
		return fmt.Errorf("failed to send trial expired email: %w", err)
	}
	return nil
}
