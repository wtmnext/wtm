package email

import (
	"crypto/tls"
	"fmt"

	"github.com/google/uuid"
	"github.com/nbittich/wtm/config"
	"gopkg.in/gomail.v2"
)

var (
	dialer   *gomail.Dialer
	MailChan = make(chan interface{}, 4)
)

func init() {
	dialer = gomail.NewDialer(config.Host, config.SMTPPort, config.SMTPFrom, config.SMTPPassword)
	if !config.SMTPSSL {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
}

func SendAsync(to []string, bcc []string, subject string, htmlBody string, attach ...string) {
	id := uuid.New()
	MailChan <- fmt.Sprintf("[%s] Sending email '%s'...", id, subject)
	m := gomail.NewMessage()
	m.SetHeader("From", config.SMTPFrom)
	m.SetHeader("To", to...)
	m.SetHeader("Cc", bcc...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)
	for _, f := range attach {
		m.Attach(f)
	}
	if err := dialer.DialAndSend(m); err != nil {
		MailChan <- err
	} else {
		MailChan <- fmt.Sprintf("[%s] mail '%s' sent", id, subject)
	}
}
