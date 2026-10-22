package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type mailer struct {
	host     string
	port     string
	username string
	password string
	from     *mail.Address
	to       *mail.Address
}

// mailerFromEnv returns nil when SMTP_HOST is unset, which disables notifications.
func mailerFromEnv() (*mailer, error) {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil, nil
	}
	m := &mailer{
		host:     host,
		port:     os.Getenv("SMTP_PORT"),
		username: os.Getenv("SMTP_USERNAME"),
		password: os.Getenv("SMTP_PASSWORD"),
	}
	if m.port == "" {
		m.port = "587"
	}

	from := os.Getenv("NOTIFY_FROM")
	if from == "" {
		from = m.username
	}
	var err error
	if m.from, err = mail.ParseAddress(from); err != nil {
		return nil, fmt.Errorf("NOTIFY_FROM: %w", err)
	}
	if m.to, err = mail.ParseAddress(os.Getenv("NOTIFY_TO")); err != nil {
		return nil, fmt.Errorf("NOTIFY_TO: %w", err)
	}
	return m, nil
}

// maxEmailsPerDay caps notification emails in any rolling 24 hours. The last
// one allowed says so, and inquiries after it are saved but not emailed.
const maxEmailsPerDay = 5

func (m *mailer) sendInquiry(c *Inquiry, last bool) error {
	subject, body := inquiryEmail(c, last)
	msg, err := m.message(subject, body, c.Email, time.Now())
	if err != nil {
		return err
	}
	return m.send(msg)
}

func inquiryEmail(c *Inquiry, last bool) (subject, body string) {
	subject = "New Weibel Painting inquiry from " + c.Name

	var b strings.Builder
	if last {
		subject = "Multiple new Weibel Painting inquiries: email notifications paused"
		fmt.Fprintf(&b, "You've received %d inquiries in the last 24 hours. To protect your inbox,\n", maxEmailsPerDay)
		b.WriteString("new inquiries are still being saved but won't be emailed until fewer\n")
		fmt.Fprintf(&b, "than %d have arrived in the past 24 hours. Check the inquiries table for\n", maxEmailsPerDay)
		b.WriteString("anything you missed.\n\n")
	}
	fmt.Fprintf(&b, "Inquiry #%d\n\n", c.ID)
	fmt.Fprintf(&b, "Name:    %s\n", c.Name)
	fmt.Fprintf(&b, "Email:   %s\n", c.Email)
	if c.Phone != "" {
		fmt.Fprintf(&b, "Phone:   %s\n", c.Phone)
	}
	if addr := formatAddress(c); addr != "" {
		fmt.Fprintf(&b, "Address: %s\n", addr)
	}
	fmt.Fprintf(&b, "\nProject details:\n%s\n", c.Message)
	return subject, b.String()
}

func formatAddress(c *Inquiry) string {
	stateZip := strings.TrimSpace(c.State + " " + c.Zip)
	var parts []string
	for _, p := range []string{c.Street, c.City, stateZip} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
}

func (m *mailer) message(subject, body, replyTo string, date time.Time) ([]byte, error) {
	var buf bytes.Buffer
	// mime.QEncoding encodes CR/LF, so user input can't inject headers.
	fmt.Fprintf(&buf, "From: %s\r\n", m.from)
	fmt.Fprintf(&buf, "To: %s\r\n", m.to)
	fmt.Fprintf(&buf, "Reply-To: %s\r\n", replyTo)
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&buf, "Date: %s\r\n", date.Format(time.RFC1123Z))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")

	qp := quotedprintable.NewWriter(&buf)
	if _, err := qp.Write([]byte(body)); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *mailer) send(msg []byte) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(m.host, m.port), 10*time.Second)
	if err != nil {
		return err
	}
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	c, err := smtp.NewClient(conn, m.host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); !ok {
		return errors.New("smtp server does not support STARTTLS")
	}
	if err := c.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
		return err
	}
	if m.username != "" {
		if err := c.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
			return err
		}
	}
	if err := c.Mail(m.from.Address); err != nil {
		return err
	}
	if err := c.Rcpt(m.to.Address); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
