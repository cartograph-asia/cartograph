package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTPConfig configures outgoing mail. An empty Host means dev mode: links
// are logged instead of sent.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string // empty: no SMTP AUTH
	Password string
	From     string // e.g. "Cartograph.asia <hello@cartograph.asia>"
}

// Mailer delivers magic links.
type Mailer interface {
	SendSignInLink(ctx context.Context, to, link string) error
}

// NewMailer returns an SMTP mailer, or the dev-mode logging mailer when
// cfg.Host is empty (ConfigFromEnv allows that only with MAIL_DEV_LOG=1).
func NewMailer(cfg SMTPConfig) Mailer {
	if cfg.Host == "" {
		slog.Warn("MAIL_DEV_LOG=1 and no SMTP_HOST: sign-in links are logged, not sent")
		return LogMailer{}
	}
	return &SMTPMailer{cfg: cfg}
}

// LogMailer logs the link at info level (dev mode).
type LogMailer struct{}

// SendSignInLink logs the link.
func (LogMailer) SendSignInLink(_ context.Context, to, link string) error {
	slog.Info("sign-in link (dev mode, not sent)", "to", to, "link", link)
	return nil
}

// SMTPMailer sends plain-text mail with net/smtp. STARTTLS is required
// unless the server is on localhost.
type SMTPMailer struct {
	cfg SMTPConfig
}

const signInSubject = "Your Cartograph sign-in link"

func signInBody(link string) string {
	return "Hello,\n\n" +
		"here is your link to sign in to Cartograph:\n\n" +
		link + "\n\n" +
		"It works once, within " + strconv.Itoa(int(LinkTTL/time.Minute)) + " minutes.\n" +
		"If you did not ask for it, you can ignore this e-mail.\n\n" +
		"Cartograph.asia – a photo for every plot of land\n"
}

// SendSignInLink sends the link to to.
func (m *SMTPMailer) SendSignInLink(ctx context.Context, to, link string) error {
	msg, err := buildMessage(m.cfg.From, to, signInSubject, signInBody(link), time.Now())
	if err != nil {
		return err
	}
	return m.send(ctx, to, msg)
}

func (m *SMTPMailer) send(ctx context.Context, to string, msg []byte) error {
	from, err := mail.ParseAddress(m.cfg.From)
	if err != nil {
		return fmt.Errorf("smtp: MAIL_FROM: %w", err)
	}
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp: dial %s: %w", addr, err)
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp: greeting: %w", err)
	}
	defer c.Close()

	// Greet with the sender's domain rather than net/smtp's "localhost",
	// which some relays refuse.
	if at := strings.LastIndexByte(from.Address, '@'); at >= 0 {
		if err := c.Hello(from.Address[at+1:]); err != nil {
			return fmt.Errorf("smtp: EHLO: %w", err)
		}
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp: starttls: %w", err)
		}
	} else if !isLocalhost(m.cfg.Host) {
		return errors.New("smtp: server does not offer STARTTLS")
	}
	if m.cfg.User != "" {
		// PlainAuth refuses to send credentials without TLS (except to localhost).
		if err := c.Auth(smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)); err != nil {
			return fmt.Errorf("smtp: auth: %w", err)
		}
	}
	if err := c.Mail(from.Address); err != nil {
		return fmt.Errorf("smtp: MAIL FROM: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp: RCPT TO: %w", err)
	}
	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp: DATA: %w", err)
	}
	if _, err := wc.Write(msg); err != nil {
		return fmt.Errorf("smtp: write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp: end of data: %w", err)
	}
	return c.Quit()
}

func isLocalhost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// buildMessage assembles a plain-text UTF-8 message. to has already been
// validated (normalizeEmail), so no header can be injected through it.
func buildMessage(from, to, subject, body string, now time.Time) ([]byte, error) {
	fromAddr, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("smtp: MAIL_FROM: %w", err)
	}
	if strings.ContainsAny(to, "\r\n") {
		return nil, errors.New("smtp: invalid recipient")
	}
	domain := "cartograph.asia"
	if at := strings.LastIndexByte(fromAddr.Address, '@'); at >= 0 {
		domain = fromAddr.Address[at+1:]
	}
	id := make([]byte, 12)
	rand.Read(id)

	var b bytes.Buffer
	h := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	h("From", fromAddr.String())
	h("To", to)
	h("Subject", mime.QEncoding.Encode("utf-8", subject))
	h("Date", now.Format(time.RFC1123Z))
	h("Message-ID", "<"+hex.EncodeToString(id)+"@"+domain+">")
	h("MIME-Version", "1.0")
	h("Content-Type", "text/plain; charset=utf-8")
	h("Content-Transfer-Encoding", "quoted-printable")
	h("Auto-Submitted", "auto-generated")
	b.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&b)
	if _, err := qp.Write([]byte(strings.ReplaceAll(body, "\n", "\r\n"))); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
