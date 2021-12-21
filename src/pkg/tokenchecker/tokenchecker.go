package tokenchecker

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TokenChecker struct {
	cfg        *TokenCheckerConfig
	httpClient *http.Client
}

type TokenCheckerConfig struct {
	UaaURL      string
	TLSConfig   *tls.Config
	UaaClient   string
	UaaPassword string
	Authority   string
}

// New returns a TokenChecker that has been
// configured with the TokenCheckerConfig.
func New(cfg *TokenCheckerConfig) *TokenChecker {
	return &TokenChecker{
		cfg: cfg,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: cfg.TLSConfig,
			},
			Timeout: 30 * time.Second,
		},
	}
}

// CheckToken verifies that the token contains
// the appropriate TokenCheckerConfig.Authority.
func (t *TokenChecker) CheckToken(token string) error {
	form := url.Values{}
	form.Set("token", token)
	form.Set("scopes", t.cfg.Authority)

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/check_token", t.cfg.UaaURL),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.cfg.UaaClient, t.cfg.UaaPassword)

	res, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("Received bad check_token status from uaa: %d %s", res.StatusCode, res.Body)
	}

	return nil
}
