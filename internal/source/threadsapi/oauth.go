package threadsapi

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
	"os/exec"
	"strings"
	"time"
)

// OAuth flow with an explicit scope list. The dashboard token generator never includes
// threads_keyword_search, so this is the only way to get a token with it.

const Scopes = "threads_basic,threads_keyword_search"

type OAuth struct {
	AppID       string
	AppSecret   string
	RedirectURI string // must be registered in the app; https://localhost:8443/callback by default
	Out         io.Writer
}

type Token struct {
	AccessToken string
	ExpiresIn   int64
}

// Authorize opens the consent page, waits for the callback and returns a long-lived token.
func (o OAuth) Authorize(ctx context.Context) (Token, error) {
	if o.AppID == "" || o.AppSecret == "" {
		return Token{}, errors.New("THREADS_APP_ID and THREADS_APP_SECRET are required")
	}
	ru, err := url.Parse(o.RedirectURI)
	if err != nil || ru.Scheme != "https" {
		return Token{}, fmt.Errorf("redirect uri must be https: %q", o.RedirectURI)
	}
	authURL := AuthorizeURL + "?" + url.Values{
		"client_id":     {o.AppID},
		"redirect_uri":  {o.RedirectURI},
		"scope":         {Scopes},
		"response_type": {"code"},
	}.Encode()

	codeCh := make(chan string, 1)
	srv, err := startCallbackServer(ru, codeCh)
	if err != nil {
		return Token{}, err
	}
	defer srv.Close()

	fmt.Fprintln(o.Out, "Opening the Threads authorization page in your default browser.")
	fmt.Fprintln(o.Out, "The browser will warn about the self-signed localhost certificate: choose Advanced -> Proceed.")
	fmt.Fprintln(o.Out, "If it did not open, paste this URL yourself:")
	fmt.Fprintln(o.Out, authURL)
	openBrowser(authURL)

	var code string
	select {
	case code = <-codeCh:
	case <-time.After(5 * time.Minute):
		return Token{}, errors.New("timeout waiting for the OAuth callback")
	case <-ctx.Done():
		return Token{}, ctx.Err()
	}
	code = strings.TrimSuffix(code, "#_")

	short, err := o.exchangeCode(ctx, code)
	if err != nil {
		return Token{}, err
	}
	return o.exchangeLongLived(ctx, short)
}

func startCallbackServer(ru *url.URL, codeCh chan<- string) (*http.Server, error) {
	cert, err := selfSignedCert()
	if err != nil {
		return nil, err
	}
	host := ru.Hostname()
	port := ru.Port()
	if port == "" {
		port = "443"
	}
	mux := http.NewServeMux()
	mux.HandleFunc(ru.Path, func(w http.ResponseWriter, r *http.Request) {
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
		Handler:   mux,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, fmt.Errorf("listen %s:%s: %w", host, port, err)
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

func (o OAuth) exchangeCode(ctx context.Context, code string) (string, error) {
	form := url.Values{
		"client_id":     {o.AppID},
		"client_secret": {o.AppSecret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {o.RedirectURI},
		"code":          {code},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.AccessToken == "" {
		return "", fmt.Errorf("code exchange failed (%d): %.300s", resp.StatusCode, body)
	}
	return r.AccessToken, nil
}

func (o OAuth) exchangeLongLived(ctx context.Context, short string) (Token, error) {
	u := BaseURL + "/access_token?" + url.Values{
		"grant_type":    {"th_exchange_token"},
		"client_secret": {o.AppSecret},
		"access_token":  {short},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Token{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.AccessToken == "" {
		return Token{}, fmt.Errorf("long-lived exchange failed (%d): %.300s", resp.StatusCode, body)
	}
	return Token{AccessToken: r.AccessToken, ExpiresIn: r.ExpiresIn}, nil
}

func openBrowser(u string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", u).Start()
}
