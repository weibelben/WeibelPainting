package main

import (
	"mime"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestInquiryEmail(t *testing.T) {
	c := Inquiry{ID: 7, Name: "Ana", Email: "ana@example.com", City: "Austin", State: "TX", Zip: "78701", Message: "Paint the kitchen"}
	subject, body := inquiryEmail(&c)
	if subject != "New Weibel Painting inquiry from Ana" {
		t.Fatalf("subject = %q", subject)
	}
	for _, want := range []string{"Inquiry #7", "ana@example.com", "Address: Austin, TX 78701", "Paint the kitchen"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "Phone:") {
		t.Errorf("body includes empty phone:\n%s", body)
	}
}

func TestMessage(t *testing.T) {
	m := &mailer{
		from: &mail.Address{Name: "Website", Address: "site@example.com"},
		to:   &mail.Address{Address: "owner@example.com"},
	}
	date := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	msg, err := m.message("New Weibel Painting inquiry from Zoë\r\nBcc: evil@example.com", "Hi Zoë\n", "ana@example.com", date)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := mail.ReadMessage(strings.NewReader(string(msg)))
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Header.Get("Bcc"); got != "" {
		t.Fatalf("header injection: Bcc = %q", got)
	}
	if got := parsed.Header.Get("Reply-To"); got != "ana@example.com" {
		t.Errorf("Reply-To = %q", got)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	if err != nil {
		t.Fatal(err)
	}
	if subject != "New Weibel Painting inquiry from Zoë\r\nBcc: evil@example.com" {
		t.Errorf("Subject = %q", subject)
	}
}
