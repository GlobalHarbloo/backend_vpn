package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"time"
)

// SendEmail отправляет multipart/alternative письмо (plain + html).
// Использует env: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS, EMAIL_FROM.
func SendEmail(to, subject, plainBody, htmlBody string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "smtp.gmail.com"
	}
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "587"
	}
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	// From header (can be "Name <email>") and envelopeFrom (plain address for SMTP MAIL FROM)
	from := os.Getenv("EMAIL_FROM")
	if from == "" {
		from = smtpUser
	}
	headerFrom := from
	envelopeFrom := smtpUser
	if parsed, err := mail.ParseAddress(from); err == nil {
		// parsed.Address is plain email for MAIL FROM, parsed.String() is properly formatted header
		envelopeFrom = parsed.Address
		headerFrom = parsed.String()
	}

	boundary := "==BOUNDARY=="
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", headerFrom))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
	buf.WriteString("\r\n")

	// Plain part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	buf.WriteString(plainBody)
	buf.WriteString("\r\n")

	// HTML part
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	buf.WriteString(htmlBody)
	buf.WriteString("\r\n")

	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	addr := net.JoinHostPort(smtpHost, smtpPort)
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	// Use implicit TLS for port 465
	if smtpPort == "465" {
		tlsConfig := &tls.Config{ServerName: smtpHost}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls dial: %w", err)
		}
		c, err := smtp.NewClient(conn, smtpHost)
		if err != nil {
			return fmt.Errorf("smtp new client: %w", err)
		}
		defer c.Close()
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
		// MAIL FROM must be a plain email address, not a display name. Use envelopeFrom.
		if err := c.Mail(envelopeFrom); err != nil {
			return err
		}
		if err := c.Rcpt(to); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(buf.Bytes()); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		_ = c.Quit()
		log.Printf("[EMAIL] Sent to %s (subject: %s)", to, subject)
		return nil
	}

	// Plain TCP + STARTTLS expected
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("tcp dial: %w", err)
	}
	c, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Close()
	if ok, _ := c.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: smtpHost}
		if err := c.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(envelopeFrom); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	_ = c.Quit()
	log.Printf("[EMAIL] Sent to %s (subject: %s)", to, subject)
	return nil
}

// SendResetCodeEmail отправляет код для сброса пароля в виде plain+HTML письма.
func SendResetCodeEmail(to, code string, expiresAt time.Time) error {
	subject := "Код для сброса пароля"
	minutes := int(time.Until(expiresAt).Minutes())
	if minutes <= 0 {
		minutes = 15
	}

	plain := fmt.Sprintf("Ваш код для сброса пароля: %s\nКод действителен %d минут.", code, minutes)

	// Simple HTML template
	tmpl := `<html><body><div style="font-family: Arial, sans-serif; max-width:600px; margin:0 auto; padding:20px; border:1px solid #eee;">
  <h2 style="color:#333;">Код для сброса пароля</h2>
  <p style="font-size:18px;"><strong style="font-size:22px;">{{.Code}}</strong></p>
  <p>Код действителен {{.Minutes}} минут. Если вы не запрашивали сброс пароля — просто проигнорируйте это письмо.</p>
  <hr/>
  <p style="font-size:12px;color:#666;">Если письмо пришло не вам, проигнорируйте его.</p>
</div></body></html>`

	data := struct {
		Code    string
		Minutes int
	}{Code: code, Minutes: minutes}
	var htmlBuf bytes.Buffer
	t, err := template.New("reset").Parse(tmpl)
	if err != nil {
		// fallback to simple html
		htmlBuf.WriteString(fmt.Sprintf("<p>Ваш код: <strong>%s</strong></p>", code))
	} else {
		if err := t.Execute(&htmlBuf, data); err != nil {
			htmlBuf.WriteString(fmt.Sprintf("<p>Ваш код: <strong>%s</strong></p>", code))
		}
	}

	return SendEmail(to, subject, plain, htmlBuf.String())
}
