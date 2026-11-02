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
	From     string // e.g. "cartograph.asia <hello@cartograph.asia>"
}

// Mailer delivers magic links, invitations and the invitation-only notice.
type Mailer interface {
	SendSignInLink(ctx context.Context, to, link string) error
	// SendInvitation sends an admin's invitation: inviter is their
	// Surveyor name, note their optional personal note, link a magic link.
	SendInvitation(ctx context.Context, to, inviter, note, link string, ttl time.Duration) error
	// SendInvitationOnly tells an address without an account or invitation
	// that sign-up needs an invitation. It contains no link.
	SendInvitationOnly(ctx context.Context, to string) error
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

// SendInvitation logs the invitation's link.
func (LogMailer) SendInvitation(_ context.Context, to, inviter, _, link string, _ time.Duration) error {
	slog.Info("invitation link (dev mode, not sent)", "to", to, "inviter", inviter, "link", link)
	return nil
}

// SendInvitationOnly logs that the notice would be sent.
func (LogMailer) SendInvitationOnly(_ context.Context, to string) error {
	slog.Info("invitation-only notice (dev mode, not sent)", "to", to)
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
		signature
}

const (
	invitationSubject     = "You're invited to Cartograph"
	invitationOnlySubject = "Cartograph is invitation-only for now"
	signature             = "cartograph.asia – a photo for every plot of land\n"
)

func invitationBody(inviter, note, link, baseURL string, ttl time.Duration) string {
	var b strings.Builder
	b.WriteString("Hello,\n\n")
	b.WriteString(inviter + " invited you to survey with Cartograph.\n\n")
	if note != "" {
		b.WriteString(note + "\n\n")
	}
	b.WriteString("Cartograph divides the land into plots of about 0.74 km² and asks for one photo of each. " +
		"The first photo of a plot claims it for the Surveyor who took it.\n\n")
	b.WriteString("Follow this link to sign in and choose your Surveyor name:\n\n" + link + "\n\n")
	b.WriteString("The link works for " + days(ttl) + ". After that, sign in at " + baseURL + "/signin with this address.\n\n")
	b.WriteString(signature)
	return b.String()
}

func days(d time.Duration) string {
	n := int(d.Round(time.Hour) / (24 * time.Hour))
	switch {
	case n == 1:
		return "1 day"
	case n > 1:
		return strconv.Itoa(n) + " days"
	}
	return strconv.Itoa(int(d.Round(time.Hour)/time.Hour)) + " hours"
}

func invitationOnlyBody() string {
	return "Hello,\n\n" +
		"someone asked to sign in to Cartograph with this address.\n\n" +
		"Cartograph is in an early test with a few invited Surveyors, so for now an account needs an invitation. " +
		"Reply to this e-mail if you would like one.\n\n" +
		"Nothing was created for this address. If you did not ask, you can ignore this e-mail.\n\n" +
		signature
}

// SendInvitation sends an invitation to to.
func (m *SMTPMailer) SendInvitation(ctx context.Context, to, inviter, note, link string, ttl time.Duration) error {
	msg, err := buildMessage(m.cfg.From, to, invitationSubject, invitationBody(inviter, note, link, baseOf(link), ttl), time.Now())
	if err != nil {
		return err
	}
	return m.send(ctx, to, msg)
}

// baseOf is the origin of a magic link.
func baseOf(link string) string {
	if i := strings.Index(link, "/signin/link"); i > 0 {
		return link[:i]
	}
	return link
}

// SendInvitationOnly sends the invitation-only notice to to.
func (m *SMTPMailer) SendInvitationOnly(ctx context.Context, to string) error {
	msg, err := buildMessage(m.cfg.From, to, invitationOnlySubject, invitationOnlyBody(), time.Now())
	if err != nil {
		return err
	}
	return m.send(ctx, to, msg)
}

// SendSignInLink sends the link to to.
func (m *SMTPMailer) SendSignInLink(ctx context.Context, to, link string) error {
	msg, err := buildMessage(m.cfg.From, to, signInSubject, signInBody(link), time.Now())
	if err != nil {
		return err
	}
	return m.send(ctx, to, msg)
}

// Check connects, greets, starts TLS and authenticates, then quits without
// sending anything: the start-up check that the SMTP settings work.
func (m *SMTPMailer) Check(ctx context.Context) error {
	c, _, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.Quit()
}

// Sender is MAIL_FROM's address, for the log.
func (m *SMTPMailer) Sender() string {
	if a, err := mail.ParseAddress(m.cfg.From); err == nil {
		return a.Address
	}
	return ""
}

func (m *SMTPMailer) send(ctx context.Context, to string, msg []byte) error {
	c, from, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
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

// connect dials, greets, starts TLS (required unless on localhost) and
// authenticates. The caller closes the client.
func (m *SMTPMailer) connect(ctx context.Context) (*smtp.Client, *mail.Address, error) {
	from, err := mail.ParseAddress(m.cfg.From)
	if err != nil {
		return nil, nil, fmt.Errorf("smtp: MAIL_FROM: %w", err)
	}
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("smtp: dial %s: %w", addr, err)
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("smtp: greeting: %w", err)
	}
	fail := func(e error) (*smtp.Client, *mail.Address, error) {
		c.Close()
		return nil, nil, e
	}

	// Greet with the sender's domain rather than net/smtp's "localhost",
	// which some relays refuse.
	if at := strings.LastIndexByte(from.Address, '@'); at >= 0 {
		if err := c.Hello(from.Address[at+1:]); err != nil {
			return fail(fmt.Errorf("smtp: EHLO: %w", err))
		}
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fail(fmt.Errorf("smtp: starttls: %w", err))
		}
	} else if !isLocalhost(m.cfg.Host) {
		return fail(errors.New("smtp: server does not offer STARTTLS"))
	}
	if m.cfg.User != "" {
		// PlainAuth refuses to send credentials without TLS (except to localhost).
		if err := c.Auth(smtp.PlainAuth("", m.cfg.User, m.cfg.Password, m.cfg.Host)); err != nil {
			return fail(fmt.Errorf("smtp: auth: %w", err))
		}
	}
	return c, from, nil
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
