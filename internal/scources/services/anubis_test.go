package services

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/youwannahackme/subix/pkg/types"
)

func TestAnubis(t *testing.T) {
	domain := "github.com"
	a := &Anubis{}
	cfg := &types.Config{
		Timeout: 15 * time.Second,
	}
	client := &http.Client{
		Timeout: cfg.Timeout,
	}
	session := &types.Session{
		Config: cfg,
		Client: client,
	}

	// Make direct request to see what it returns
	url := "https://anubisdb.com/subdomains/" + domain
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/115.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	t.Logf("Status Code: %d", resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Body length: %d", len(bodyBytes))

	// Run method
	subs, err := a.Run(domain, session)
	if err != nil {
		t.Fatalf("Anubis.Run failed: %v", err)
	}
	t.Logf("Found %d subdomains: %v", len(subs), subs)
}
