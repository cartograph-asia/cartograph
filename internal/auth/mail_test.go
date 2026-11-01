package auth

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeSMTP accepts one message on localhost (no TLS, which SMTPMailer only
// allows for localhost) and returns the transcript of client commands.
func fakeSMTP(t *testing.T) (port int, got <-chan string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	ch := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(5 * time.Second))
		r := bufio.NewReader(c)
		say := func(s string) { _, _ = c.Write([]byte(s + "\r\n")) }
		var log strings.Builder
		say("220 fake ESMTP")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				ch <- log.String()
				return
			}
			log.WriteString(line)
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"):
				say("250-fake")
				say("250 AUTH PLAIN")
			case strings.HasPrefix(cmd, "AUTH"), strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
				if strings.HasPrefix(cmd, "AUTH") {
					say("235 ok")
				} else {
					say("250 ok")
				}
			case cmd == "DATA":
				say("354 go on")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						ch <- log.String()
						return
					}
					log.WriteString(l)
					if l == ".\r\n" {
						break
					}
				}
				say("250 queued")
			case cmd == "QUIT":
				say("221 bye")
				ch <- log.String()
				return
			default:
				say("500 what")
			}
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, ch
}

func TestSMTPMailer(t *testing.T) {
	port, got := fakeSMTP(t)
	m := NewMailer(SMTPConfig{Host: "127.0.0.1", Port: port, User: "u", Password: "p", From: "Cartograph.asia <hello@cartograph.asia>"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.SendSignInLink(ctx, "ada@example.org", "http://localhost:8080/signin/link?token=abc"); err != nil {
		t.Fatal(err)
	}
	s := <-got
	for _, w := range []string{"EHLO cartograph.asia", "AUTH PLAIN", "MAIL FROM:<hello@cartograph.asia>", "RCPT TO:<ada@example.org>", "token=3Dabc", "QUIT"} {
		if !strings.Contains(s, w) {
			t.Errorf("transcript lacks %q:\n%s", w, s)
		}
	}
}

// Plain-text SMTP (no STARTTLS) is only accepted for these hosts.
func TestIsLocalhost(t *testing.T) {
	if isLocalhost("smtp.example.org") || isLocalhost("10.0.0.1") || !isLocalhost("localhost") || !isLocalhost("127.0.0.1") || !isLocalhost("::1") {
		t.Fatal("isLocalhost")
	}
}
