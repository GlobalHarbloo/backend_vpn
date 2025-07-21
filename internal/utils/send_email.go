package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

func SendResetEmail(to, link string) error {
	smtpHost := os.Getenv("smtp.gmail.com")
	smtpPort := os.Getenv("465")
	smtpUser := os.Getenv("glebsapronov12@gmail.com")
	smtpPass := os.Getenv("")

	from := smtpUser
	subject := "Сброс пароля для вашего VPN-аккаунта"
	body := fmt.Sprintf("Для сброса пароля перейдите по ссылке:\n\n%s\n\nЕсли вы не запрашивали сброс, просто проигнорируйте это письмо.", link)
	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	addr := smtpHost + ":" + smtpPort
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
	if err != nil {
		log.Printf("[EMAIL] Ошибка отправки письма: %v", err)
		return err
	}
	log.Printf("[EMAIL] Сброс пароля отправлен на %s", to)
	return nil
} 