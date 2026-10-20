package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

var turnstileClient = &http.Client{Timeout: 10 * time.Second}

func verifyTurnstile(ctx context.Context, secret, token, ip string) (bool, error) {
	if token == "" {
		return false, nil
	}

	form := url.Values{"secret": {secret}, "response": {token}}
	if ip != "" {
		form.Set("remoteip", ip)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileURL, strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := turnstileClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("siteverify status %d", resp.StatusCode)
	}

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Success, nil
}
