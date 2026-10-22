package main

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type Customer struct {
	Email     string    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"not null"`
	Inquiries []Inquiry `gorm:"foreignKey:Email;references:Email"`
}

type Inquiry struct {
	ID    uint   `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"not null"`
	Name   string  `gorm:"not null"`
	Email   string  `gorm:"not null;index"`
	Phone   string
	Street  string
	City   string
	State   string
	Zip    string
	Message  string
}

var errInvalid = errors.New("invalid submission")

type field struct {
	value  string
	max   int
	required bool
}

func validate(c *Inquiry) error {
	c.Name = strings.TrimSpace(c.Name)
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	c.Phone = strings.TrimSpace(c.Phone)
	c.Street = strings.TrimSpace(c.Street)
	c.City = strings.TrimSpace(c.City)
	c.State = strings.ToUpper(strings.TrimSpace(c.State))
	c.Zip = strings.TrimSpace(c.Zip)
	c.Message = strings.TrimSpace(c.Message)

	fields := []field{
		{c.Name, 199, true},
		{c.Email, 254, true},
		{c.Phone, 40, false},
		{c.Street, 200, false},
		{c.City, 100, false},
		{c.State, 2, false},
		{c.Zip, 5, false},
		{c.Message, 4000, true},
	}
	for _, f := range fields {
		if f.required && f.value == "" {
			return errInvalid
		}
		if utf8.RuneCountInString(f.value) > f.max {
			return errInvalid
		}
	}

	// We only serve Washington, so the form's state field is fixed to WA.
	if c.State != "" && c.State != "WA" {
		return errInvalid
	}
	c.State = "WA"
	if c.Zip != "" && !isDigits(c.Zip, 5) {
		return errInvalid
	}

	addr, err := mail.ParseAddress(c.Email)
	if err != nil || addr.Address != c.Email {
		return errInvalid
	}
	// net/mail allows dotless domains like "user@localhost"; require a TLD.
	domain := c.Email[strings.LastIndexByte(c.Email, '@')+1:]
	if dot := strings.LastIndexByte(domain, '.'); dot <= 0 || dot == len(domain)-1 {
		return errInvalid
	}
	return nil
}

func isDigits(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
