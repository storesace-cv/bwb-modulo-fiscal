package smtp_test

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/notify/smtp"
)

func TestValidateRequiresImplicit465(t *testing.T) {
	t.Parallel()
	cfg := smtp.Config{
		Host: "smtp.example", Port: 587, Username: "u", Password: "p",
		TLSMode: "starttls", FromAddress: "a@example.com", AdminNotifyAddress: "b@example.com",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for non-465/starttls")
	}
	cfg.Port = 465
	cfg.TLSMode = "implicit"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
}

func TestSendAdminTestOverFakeImplicitTLS(t *testing.T) {
	t.Parallel()
	srv := startFakeImplicitSMTP(t, "user", "secret")
	defer srv.Close()

	cfg := smtp.Config{
		Host: "127.0.0.1", Port: 465, Username: "user", Password: "secret",
		TLSMode: "implicit", FromAddress: "noreply@example.com", FromName: "BWB Fiscal",
		AdminNotifyAddress: "admin@example.com",
	}
	mailer, err := smtp.NewMailerWithDialer(cfg, func(ctx context.Context, network, addr string, tlsCfg *tls.Config) (net.Conn, error) {
		d := tls.Dialer{Config: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}, NetDialer: &net.Dialer{Timeout: 5 * time.Second}}
		return d.DialContext(ctx, "tcp", srv.Addr())
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := mailer.SendAdminTest(context.Background(), "req_test_1")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if st.Status != "sent" {
		t.Fatalf("status=%s reason=%s", st.Status, st.Reason)
	}
	if st.ToDomain != "example.com" || st.Port != 465 || st.TLSMode != "implicit" {
		t.Fatalf("unexpected status %+v", st)
	}
	msg := srv.LastMessage()
	if !strings.Contains(msg, "Subject: BWB Fiscal") {
		t.Fatalf("missing subject in message: %q", msg)
	}
	low := strings.ToLower(msg)
	for _, banned := range []string{"secret", "password=", "smtp_password", "nif="} {
		if strings.Contains(low, banned) {
			t.Fatalf("message leaked %q: %q", banned, msg)
		}
	}
}

func TestSendAdminAlertDigestOverFakeImplicitTLS(t *testing.T) {
	t.Parallel()
	srv := startFakeImplicitSMTP(t, "user", "secret")
	defer srv.Close()

	cfg := smtp.Config{
		Host: "127.0.0.1", Port: 465, Username: "user", Password: "secret",
		TLSMode: "implicit", FromAddress: "noreply@example.com", FromName: "BWB Fiscal",
		AdminNotifyAddress: "admin@example.com",
	}
	mailer, err := smtp.NewMailerWithDialer(cfg, func(ctx context.Context, network, addr string, tlsCfg *tls.Config) (net.Conn, error) {
		d := tls.Dialer{Config: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}, NetDialer: &net.Dialer{Timeout: 5 * time.Second}}
		return d.DialContext(ctx, "tcp", srv.Addr())
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := mailer.SendAdminAlertDigest(context.Background(), "req_digest_1", []smtp.AlertLine{{
		Code: "ops_retry_backlog", Severity: "warning", Message: "fila retry=5 — monitorizar tentativas",
	}})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if st.Status != "sent" {
		t.Fatalf("status=%s reason=%s", st.Status, st.Reason)
	}
	msg := srv.LastMessage()
	if !strings.Contains(msg, "digest de alertas") {
		t.Fatalf("missing digest subject/body: %q", msg)
	}
	if !strings.Contains(msg, "ops_retry_backlog") || !strings.Contains(msg, "alert_count=1") {
		t.Fatalf("missing alert code: %q", msg)
	}
	low := strings.ToLower(msg)
	for _, banned := range []string{"secret", "password=", "nif=", "dsn="} {
		if strings.Contains(low, banned) {
			t.Fatalf("message leaked %q: %q", banned, msg)
		}
	}
}

func TestSanitizeFailureDoesNotEchoPassword(t *testing.T) {
	t.Parallel()
	cfg := smtp.Config{
		Host: "127.0.0.1", Port: 465, Username: "user", Password: "super-secret-pass",
		TLSMode: "implicit", FromAddress: "a@example.com", AdminNotifyAddress: "b@example.com",
	}
	mailer, err := smtp.NewMailerWithDialer(cfg, func(ctx context.Context, network, addr string, tlsCfg *tls.Config) (net.Conn, error) {
		return nil, context.DeadlineExceeded
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := mailer.SendAdminTest(context.Background(), "req_x")
	if err == nil {
		t.Fatal("expected error")
	}
	if st.Status != "failed" || st.Reason != "smtp_timeout" {
		t.Fatalf("got %+v", st)
	}
	if strings.Contains(st.Reason, "super-secret") || strings.Contains(err.Error(), "super-secret") {
		// err.Error may contain dialer noise but password must not appear from our sanitize path in Status.Reason
		t.Fatalf("password leaked in status reason")
	}
}

type fakeSMTP struct {
	ln   net.Listener
	mu   sync.Mutex
	last string
	user string
	pass string
}

func (f *fakeSMTP) Addr() string { return f.ln.Addr().String() }
func (f *fakeSMTP) Close()       { _ = f.ln.Close() }
func (f *fakeSMTP) LastMessage() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last
}

func startFakeImplicitSMTP(t *testing.T, user, pass string) *fakeSMTP {
	t.Helper()
	cert := mustTestCert(t)
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{ln: ln, user: user, pass: pass}
	go f.serve()
	return f
}

func (f *fakeSMTP) serve() {
	for {
		c, err := f.ln.Accept()
		if err != nil {
			return
		}
		go f.handle(c)
	}
}

func (f *fakeSMTP) handle(c net.Conn) {
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))
	r := bufio.NewReader(c)
	w := bufio.NewWriter(c)
	writeLine := func(s string) {
		_, _ = w.WriteString(s + "\r\n")
		_ = w.Flush()
	}
	writeLine("220 fake-smtp ready")
	var data strings.Builder
	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if inData {
			if line == "." {
				f.mu.Lock()
				f.last = data.String()
				f.mu.Unlock()
				inData = false
				writeLine("250 OK")
				continue
			}
			data.WriteString(line)
			data.WriteByte('\n')
			continue
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO") || strings.HasPrefix(upper, "HELO"):
			writeLine("250-localhost")
			writeLine("250-AUTH PLAIN LOGIN")
			writeLine("250 OK")
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			writeLine("235 OK")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			writeLine("250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			writeLine("250 OK")
		case upper == "DATA":
			inData = true
			data.Reset()
			writeLine("354 End data with <CR><LF>.<CR><LF>")
		case upper == "QUIT":
			writeLine("221 Bye")
			return
		default:
			writeLine("250 OK")
		}
	}
}

func mustTestCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "fake-smtp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	_ = io.Discard
	return cert
}
