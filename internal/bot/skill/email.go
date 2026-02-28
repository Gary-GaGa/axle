package skill

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// EmailConfig holds SMTP/IMAP server settings for email operations.
type EmailConfig struct {
	SMTPHost string
	SMTPPort string
	IMAPHost string
	IMAPPort string
	Address  string
	Password string
}

// IsConfigured returns true if minimal email settings are present.
func (c EmailConfig) IsConfigured() bool {
	return c.Address != "" && c.Password != "" && c.SMTPHost != ""
}

// EmailSummary holds a brief view of an email message.
type EmailSummary struct {
	From    string
	Subject string
	Date    string
}

func (e EmailSummary) String() string {
	return fmt.Sprintf("📧 *%s*\n   👤 %s\n   📅 %s", e.Subject, e.From, e.Date)
}

// SendEmail sends an email via SMTP.
func SendEmail(cfg EmailConfig, to, subject, body string) error {
	if !cfg.IsConfigured() {
		return fmt.Errorf("Email 尚未設定，請先設定 Email 憑證（環境變數或 credentials.json）")
	}

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		cfg.Address, to, subject, body,
	)

	// Port 465 = implicit TLS
	if cfg.SMTPPort == "465" {
		return sendTLS(cfg, to, msg)
	}

	// Port 587 = STARTTLS (default)
	addr := cfg.SMTPHost + ":" + cfg.SMTPPort
	auth := smtp.PlainAuth("", cfg.Address, cfg.Password, cfg.SMTPHost)
	return smtp.SendMail(addr, auth, cfg.Address, []string{to}, []byte(msg))
}

func sendTLS(cfg EmailConfig, to, msg string) error {
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		cfg.SMTPHost+":"+cfg.SMTPPort,
		&tls.Config{ServerName: cfg.SMTPHost},
	)
	if err != nil {
		return fmt.Errorf("TLS 連線失敗: %w", err)
	}

	client, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("SMTP 用戶端建立失敗: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", cfg.Address, cfg.Password, cfg.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("認證失敗: %w", err)
	}
	if err := client.Mail(cfg.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = w.Write([]byte(msg)); err != nil {
		return err
	}
	return w.Close()
}

// ReadEmails fetches the last N email headers from IMAP INBOX.
func ReadEmails(cfg EmailConfig, count int) ([]EmailSummary, error) {
	if cfg.IMAPHost == "" || cfg.Address == "" || cfg.Password == "" {
		return nil, fmt.Errorf("IMAP 尚未設定，請先設定 Email 憑證")
	}

	addr := cfg.IMAPHost + ":" + cfg.IMAPPort
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp", addr,
		&tls.Config{ServerName: cfg.IMAPHost},
	)
	if err != nil {
		return nil, fmt.Errorf("IMAP 連線失敗: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	r := bufio.NewReader(conn)
	// Read server greeting
	if _, err := r.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("IMAP 讀取失敗: %w", err)
	}

	writeLine := func(tag, cmd string) error {
		_, err := fmt.Fprintf(conn, "%s %s\r\n", tag, cmd)
		return err
	}

	readUntil := func(tag string) ([]string, error) {
		var lines []string
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return lines, err
			}
			lines = append(lines, strings.TrimRight(line, "\r\n"))
			if strings.HasPrefix(line, tag+" ") {
				return lines, nil
			}
		}
	}

	// LOGIN
	if err := writeLine("a1", fmt.Sprintf("LOGIN %s %s", cfg.Address, cfg.Password)); err != nil {
		return nil, err
	}
	resp, err := readUntil("a1")
	if err != nil {
		return nil, fmt.Errorf("IMAP LOGIN 失敗: %w", err)
	}
	if !strings.Contains(resp[len(resp)-1], "OK") {
		return nil, fmt.Errorf("IMAP 認證失敗: %s", resp[len(resp)-1])
	}

	// SELECT INBOX
	if err := writeLine("a2", "SELECT INBOX"); err != nil {
		return nil, err
	}
	resp, err = readUntil("a2")
	if err != nil {
		return nil, fmt.Errorf("IMAP SELECT 失敗: %w", err)
	}

	// Parse EXISTS count
	total := 0
	for _, line := range resp {
		if strings.Contains(line, "EXISTS") {
			fmt.Sscanf(line, "* %d EXISTS", &total)
		}
	}
	if total == 0 {
		// LOGOUT
		writeLine("a9", "LOGOUT")
		return nil, fmt.Errorf("信箱為空")
	}

	// Fetch headers of last N messages
	start := total - count + 1
	if start < 1 {
		start = 1
	}
	fetchCmd := fmt.Sprintf("FETCH %d:%d (BODY[HEADER.FIELDS (FROM SUBJECT DATE)])", start, total)
	if err := writeLine("a3", fetchCmd); err != nil {
		return nil, err
	}
	resp, err = readUntil("a3")
	if err != nil {
		return nil, fmt.Errorf("IMAP FETCH 失敗: %w", err)
	}

	// Parse headers from response
	var summaries []EmailSummary
	var cur EmailSummary
	inHeader := false
	for _, line := range resp {
		if strings.Contains(line, "BODY[HEADER.FIELDS") {
			inHeader = true
			cur = EmailSummary{}
			continue
		}
		if inHeader {
			lower := strings.ToLower(line)
			switch {
			case strings.HasPrefix(lower, "from:"):
				cur.From = strings.TrimSpace(line[5:])
			case strings.HasPrefix(lower, "subject:"):
				cur.Subject = strings.TrimSpace(line[8:])
			case strings.HasPrefix(lower, "date:"):
				cur.Date = strings.TrimSpace(line[5:])
			case line == ")" || line == "":
				if cur.Subject != "" || cur.From != "" {
					summaries = append(summaries, cur)
				}
				inHeader = false
			}
		}
	}

	// LOGOUT
	writeLine("a9", "LOGOUT")

	// Reverse to show newest first
	for i, j := 0, len(summaries)-1; i < j; i, j = i+1, j-1 {
		summaries[i], summaries[j] = summaries[j], summaries[i]
	}

	return summaries, nil
}
