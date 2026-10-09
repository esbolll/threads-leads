package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

// OAuth flow for a Threads user token with an explicit scope list.
// Needs THREADS_APP_ID and THREADS_APP_SECRET in env or .env, and
// https://localhost:8443/callback registered as a redirect URI in the app.

const (
	authRedirect = "https://localhost:8443/callback"
	authScopes   = "threads_basic,threads_keyword_search"
)

func envOrDotenv(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	data, err := os.ReadFile(".env")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.Trim(strings.TrimPrefix(line, key+"="), `"'`)
		}
	}
	return ""
}

func runAuth() error {
	appID := envOrDotenv("THREADS_APP_ID")
	secret := envOrDotenv("THREADS_APP_SECRET")
	if appID == "" || secret == "" {
		return errors.New("set THREADS_APP_ID and THREADS_APP_SECRET in .env")
	}

	authURL := "https://threads.net/oauth/authorize?" + url.Values{
		"client_id":     {appID},
		"redirect_uri":  {authRedirect},
		"scope":         {authScopes},
		"response_type": {"code"},
	}.Encode()

	codeCh := make(chan string, 1)
	srv, err := startCallbackServer(codeCh)
	if err != nil {
		return err
	}
	defer srv.Close()

	fmt.Println("Opening the Threads authorization page in your default browser.")
	fmt.Println("Chrome will warn about the self-signed localhost certificate: choose Advanced -> Proceed.")
	fmt.Println("If the browser did not open, paste this URL yourself:")
	fmt.Println(authURL)
	openBrowser(authURL)

	var code string
	select {
	case code = <-codeCh:
	case <-time.After(5 * time.Minute):
		return errors.New("timeout waiting for the OAuth callback")
	}
	code = strings.TrimSuffix(code, "#_") // Meta appends #_ to the code

	shortTok, err := exchangeCode(appID, secret, code)
	if err != nil {
		return err
	}
	longTok, expires, err := exchangeLongLived(secret, shortTok)
	if err != nil {
		return err
	}
	if err := saveEnvToken(longTok); err != nil {
		return err
	}
	fmt.Printf("Long-lived token saved to .env (expires in %d days).\n", expires/86400)
	return nil
}

func startCallbackServer(codeCh chan<- string) (*http.Server, error) {
	cert, err := selfSignedCert()
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			fmt.Fprintf(w, "Authorization failed: %s %s", e, q.Get("error_description"))
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "no code in callback", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, "Authorized. You can close this tab and return to the terminal.")
		select {
		case codeCh <- code:
		default:
		}
	})
	srv := &http.Server{
		Addr:      "127.0.0.1:8443",
		Handler:   mux,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", srv.Addr, err)
	}
	go func() { _ = srv.ServeTLS(ln, "", "") }()
	return srv, nil
}

func selfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

func exchangeCode(appID, secret, code string) (string, error) {
	resp, err := httpClient.PostForm("https://graph.threads.net/oauth/access_token", url.Values{
		"client_id":     {appID},
		"client_secret": {secret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {authRedirect},
		"code":          {code},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r struct {
		AccessToken string    `json:"access_token"`
		Error       *apiError `json:"error"`
		ErrMsg      string    `json:"error_message"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.AccessToken == "" {
		return "", fmt.Errorf("code exchange failed (%d): %s", resp.StatusCode, truncate(string(body), 300))
	}
	return r.AccessToken, nil
}

func exchangeLongLived(secret, shortTok string) (string, int64, error) {
	u := apiBase + "/access_token?" + url.Values{
		"grant_type":    {"th_exchange_token"},
		"client_secret": {secret},
		"access_token":  {shortTok},
	}.Encode()
	resp, err := httpClient.Get(u)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.AccessToken == "" {
		return "", 0, fmt.Errorf("long-lived exchange failed (%d): %s", resp.StatusCode, truncate(string(body), 300))
	}
	return r.AccessToken, r.ExpiresIn, nil
}

// saveEnvToken rewrites THREADS_TOKEN in .env, keeping other lines.
func saveEnvToken(tok string) error {
	data, _ := os.ReadFile(".env")
	var out []string
	replaced := false
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "THREADS_TOKEN=") {
			out = append(out, "THREADS_TOKEN="+tok)
			replaced = true
			continue
		}
		out = append(out, line)
	}
	if !replaced {
		out = append(out, "THREADS_TOKEN="+tok)
	}
	return os.WriteFile(".env", []byte(strings.TrimRight(strings.Join(out, "\n"), "\n")+"\n"), 0o600)
}

func openBrowser(u string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", u).Start()
}
