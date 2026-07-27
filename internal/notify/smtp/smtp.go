// Package smtp provides implicit-TLS (port 465) outbound mail for admin notifications.
// Secrets must never be logged or returned in API/CLI responses.
package smtp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	envHost          = "FISCAL_SMTP_HOST"
	envPort          = "FISCAL_SMTP_PORT"
	envUsername      = "FISCAL_SMTP_USERNAME"
	envPassword      = "FISCAL_SMTP_PASSWORD"
	envTLSMode       = "FISCAL_SMTP_TLS_MODE"
	envFromAddress   = "FISCAL_SMTP_FROM_ADDRESS"
	envFromName      = "FISCAL_SMTP_FROM_NAME"
	envAdminNotifyTo = "FISCAL_SMTP_ADMIN_NOTIFICATION_EMAIL"

	tlsModeImplicit = "implicit"
	requiredPort    = 465
)

// Config is validated SMTP settings (implicit TLS only).
type Config struct {
	Host               string
	Port               int
	Username           string
	Password           string
	TLSMode            string
	FromAddress        string
	FromName           string
	AdminNotifyAddress string
}

// DeliveryStatus is a sanitized result safe for logs/API (no secrets, no raw SMTP chatter).
type DeliveryStatus struct {
	Status    string `json:"status"` // sent|failed|not_configured
	Reason    string `json:"reason,omitempty"`
	ToDomain  string `json:"to_domain,omitempty"`
	TLSMode   string `json:"tls_mode,omitempty"`
	Port      int    `json:"port,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// Mailer sends notification messages.
type Mailer interface {
	Configured() bool
	SendAdminTest(ctx context.Context, requestID string) (DeliveryStatus, error)
}

type clientMailer struct {
	cfg    Config
	dialer DialFunc
}

// DialFunc opens a TLS network connection to host:port (tests inject a fake).
type DialFunc func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error)

// EnvKeys lists SMTP environment variable names (no values).
func EnvKeys() []string {
	return []string{envHost, envPort, envUsername, envPassword, envTLSMode, envFromAddress, envFromName, envAdminNotifyTo}
}

// LoadConfigFromEnv reads optional SMTP config. Empty host → not configured (ok).
func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		Host:               strings.TrimSpace(os.Getenv(envHost)),
		Username:           strings.TrimSpace(os.Getenv(envUsername)),
		Password:           os.Getenv(envPassword), // keep exact; do not Trim (passwords may have spaces)
		TLSMode:            strings.TrimSpace(os.Getenv(envTLSMode)),
		FromAddress:        strings.TrimSpace(os.Getenv(envFromAddress)),
		FromName:           strings.TrimSpace(os.Getenv(envFromName)),
		AdminNotifyAddress: strings.TrimSpace(os.Getenv(envAdminNotifyTo)),
	}
	portRaw := strings.TrimSpace(os.Getenv(envPort))
	if cfg.Host == "" && portRaw == "" && cfg.Username == "" && cfg.Password == "" &&
		cfg.TLSMode == "" && cfg.FromAddress == "" && cfg.AdminNotifyAddress == "" {
		return Config{}, nil
	}
	if cfg.Host == "" || portRaw == "" || cfg.Username == "" || cfg.Password == "" ||
		cfg.TLSMode == "" || cfg.FromAddress == "" || cfg.AdminNotifyAddress == "" {
		return Config{}, errors.New("smtp config incomplete")
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		return Config{}, errors.New("smtp port invalid")
	}
	cfg.Port = port
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	cfg.TLSMode = normalizeTLSMode(cfg.TLSMode)
	return cfg, nil
}

func normalizeTLSMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case tlsModeImplicit, "implicit_tls":
		return tlsModeImplicit
	default:
		return ""
	}
}

// Validate enforces implicit TLS on 465 and valid addresses.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return errors.New("smtp host required")
	}
	if c.Port != requiredPort {
		return errors.New("smtp port must be 465")
	}
	if normalizeTLSMode(c.TLSMode) == "" {
		return errors.New("smtp tls mode must be implicit")
	}
	if _, err := mail.ParseAddress(c.FromAddress); err != nil {
		return errors.New("smtp from address invalid")
	}
	if _, err := mail.ParseAddress(c.AdminNotifyAddress); err != nil {
		return errors.New("smtp admin notification address invalid")
	}
	if strings.TrimSpace(c.Username) == "" || c.Password == "" {
		return errors.New("smtp credentials required")
	}
	return nil
}

// NewMailer builds a production mailer (nil if not configured).
func NewMailer(cfg Config) (Mailer, error) {
	if cfg.Host == "" {
		return &clientMailer{}, nil
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.TLSMode = tlsModeImplicit
	return &clientMailer{cfg: cfg, dialer: defaultDial}, nil
}

// NewMailerWithDialer is for tests with a fake TLS SMTP listener.
func NewMailerWithDialer(cfg Config, dial DialFunc) (Mailer, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if dial == nil {
		return nil, errors.New("dialer required")
	}
	cfg.TLSMode = tlsModeImplicit
	return &clientMailer{cfg: cfg, dialer: dial}, nil
}

func (m *clientMailer) Configured() bool {
	return m != nil && m.cfg.Host != ""
}

func (m *clientMailer) SendAdminTest(ctx context.Context, requestID string) (DeliveryStatus, error) {
	if !m.Configured() {
		return DeliveryStatus{Status: "not_configured", Reason: "smtp_not_configured", RequestID: requestID}, nil
	}
	toDomain := domainOf(m.cfg.AdminNotifyAddress)
	base := DeliveryStatus{
		TLSMode: tlsModeImplicit, Port: requiredPort, ToDomain: toDomain, RequestID: requestID,
	}
	subject := "BWB Fiscal — teste de notificação"
	body := strings.Join([]string{
		"Este é um email de teste autorizado do módulo fiscal BWB.",
		"Não contém passwords, tokens, NIF, DSN nem credenciais AGT.",
		"request_id=" + sanitizeToken(requestID),
		"tls_mode=implicit",
		"port=465",
	}, "\n")
	if err := m.send(ctx, m.cfg.AdminNotifyAddress, subject, body); err != nil {
		base.Status = "failed"
		base.Reason = sanitizeErr(err)
		return base, err
	}
	base.Status = "sent"
	return base, nil
}

func (m *clientMailer) send(ctx context.Context, to, subject, body string) error {
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	tlsCfg := &tls.Config{
		ServerName: m.cfg.Host,
		MinVersion: tls.VersionTLS12,
	}
	conn, err := m.dialer(ctx, "tcp", addr, tlsCfg)
	if err != nil {
		return err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	}

	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	if ok, _ := c.Extension("AUTH"); !ok {
		return errors.New("smtp auth extension required")
	}
	if err := c.Auth(auth); err != nil {
		return err
	}
	from := m.cfg.FromAddress
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	msg := buildMessage(m.cfg.FromName, from, to, subject, body)
	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func defaultDial(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
	d := tls.Dialer{Config: cfg, NetDialer: &net.Dialer{Timeout: 15 * time.Second}}
	return d.DialContext(ctx, network, addr)
}

func buildMessage(fromName, from, to, subject, body string) string {
	fromHeader := from
	if strings.TrimSpace(fromName) != "" {
		fromHeader = fmt.Sprintf("%s <%s>", sanitizeHeader(fromName), from)
	}
	var b strings.Builder
	b.WriteString("From: " + fromHeader + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return b.String()
}

func domainOf(addr string) string {
	a, err := mail.ParseAddress(addr)
	if err != nil {
		return "invalid"
	}
	parts := strings.Split(a.Address, "@")
	if len(parts) != 2 {
		return "invalid"
	}
	return strings.ToLower(parts[1])
}

func sanitizeHeader(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, s)
}

func sanitizeToken(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "none"
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			return r
		default:
			return -1
		}
	}, s)
}

func sanitizeErr(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
		return "smtp_timeout"
	case strings.Contains(msg, "auth"):
		return "smtp_auth_failed"
	case strings.Contains(msg, "tls") || strings.Contains(msg, "certificate"):
		return "smtp_tls_failed"
	case strings.Contains(msg, "connection refused") || strings.Contains(msg, "connect"):
		return "smtp_connect_failed"
	default:
		return "smtp_send_failed"
	}
}
