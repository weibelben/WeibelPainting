package main

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type Customer struct {
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

func validate(c *Customer) error {
	c.Name = strings.TrimSpace(c.Name)
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	c.Phone = strings.TrimSpace(c.Phone)
	c.Street = strings.TrimSpace(c.Street)
	c.City = strings.TrimSpace(c.City)
	c.State = strings.TrimSpace(c.State)
	c.Zip = strings.TrimSpace(c.Zip)
	c.Message = strings.TrimSpace(c.Message)

	fields := []field{
		{c.Name, 200, true},
		{c.Email, 254, true},
		{c.Phone, 40, false},
		{c.Street, 200, false},
		{c.City, 100, false},
		{c.State, 50, false},
		{c.Zip, 20, false},
		{c.Message, 4000, false},
	}
	for _, f := range fields {
		if f.required && f.value == "" {
			return errInvalid
		}
		if utf8.RuneCountInString(f.value) > f.max {
			return errInvalid
		}
	}

	addr, err := mail.ParseAddress(c.Email)
	if err != nil || addr.Address != c.Email {
		return errInvalid
	}
	return nil
}
