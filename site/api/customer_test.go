package main

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      Customer
		wantErr bool
	}{
		{"valid minimal", Customer{Name: "Ana", Email: "ana@example.com", Message: "Paint the kitchen"}, false},
		{"trims and lowercases", Customer{Name: " Ana ", Email: " Ana@Example.com ", Message: "Paint the kitchen"}, false},
		{"missing name", Customer{Email: "ana@example.com", Message: "Paint the kitchen"}, true},
		{"missing email", Customer{Name: "Ana", Message: "Paint the kitchen"}, true},
		{"bad email", Customer{Name: "Ana", Email: "not-an-email", Message: "Paint the kitchen"}, true},
		{"bad email 2", Customer{Name: "Ana", Email: "not-an-email@asdf", Message: "Paint the kitchen"}, true},
		{"trailing dot domain", Customer{Name: "Ana", Email: "ana@example.", Message: "Paint the kitchen"}, true},
		{"subdomain email", Customer{Name: "Ana", Email: "ana@mail.example.co.uk", Message: "Paint the kitchen"}, false},
		{"display name email", Customer{Name: "Ana", Email: "Ana <ana@example.com>", Message: "Paint the kitchen"}, true},
		{"missing message", Customer{Name: "Ana", Email: "ana@example.com"}, true},
		{"blank message", Customer{Name: "Ana", Email: "ana@example.com", Message: "   "}, true},
		{"name at max", Customer{Name: strings.Repeat("a", 199), Email: "ana@example.com", Message: "Paint the kitchen"}, false},
		{"name too long", Customer{Name: strings.Repeat("a", 200), Email: "ana@example.com", Message: "Paint the kitchen"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.in
			err := validate(&c)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateNormalizes(t *testing.T) {
	c := Customer{Name: " Ana ", Email: " Ana@Example.com ", Message: "Paint the kitchen"}
	if err := validate(&c); err != nil {
		t.Fatal(err)
	}
	if c.Name != "Ana" || c.Email != "ana@example.com" {
		t.Fatalf("got name %q email %q", c.Name, c.Email)
	}
}
