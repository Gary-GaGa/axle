package skill

import (
	"context"
	"testing"
)

func TestGHCheckInstalled(t *testing.T) {
	// gh should be installed on this system
	err := GHCheckInstalled()
	if err != nil {
		t.Skipf("gh CLI not installed: %v", err)
	}
}

func TestGHTimeout(t *testing.T) {
	timeout := GHTimeout()
	if timeout <= 0 {
		t.Errorf("expected positive timeout, got %v", timeout)
	}
}

func TestGHPRList_InvalidWorkspace(t *testing.T) {
	ctx := context.Background()
	_, err := GHPRList(ctx, "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid workspace")
	}
}

func TestGHIssueList_InvalidWorkspace(t *testing.T) {
	ctx := context.Background()
	_, err := GHIssueList(ctx, "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid workspace")
	}
}

func TestGHCIStatus_InvalidWorkspace(t *testing.T) {
	ctx := context.Background()
	_, err := GHCIStatus(ctx, "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid workspace")
	}
}

func TestGHRepoView_InvalidWorkspace(t *testing.T) {
	ctx := context.Background()
	_, err := GHRepoView(ctx, "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid workspace")
	}
}

func TestGHPRView_InvalidWorkspace(t *testing.T) {
	ctx := context.Background()
	_, err := GHPRView(ctx, "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for invalid workspace")
	}
}

func TestGHIssueView_InvalidWorkspace(t *testing.T) {
	ctx := context.Background()
	_, err := GHIssueView(ctx, "/nonexistent/path/xyz", "1")
	if err == nil {
		t.Fatal("expected error for invalid workspace")
	}
}

func TestGHPRCreate_InvalidWorkspace(t *testing.T) {
	ctx := context.Background()
	_, err := GHPRCreate(ctx, "/nonexistent/path", "title", "body")
	if err == nil {
		t.Fatal("expected error for invalid workspace")
	}
}

func TestEmailConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  EmailConfig
		want bool
	}{
		{"empty", EmailConfig{}, false},
		{"partial", EmailConfig{Address: "a@b.com"}, false},
		{"full", EmailConfig{Address: "a@b.com", Password: "p", SMTPHost: "smtp.test.com"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsConfigured()
			if got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendEmail_NotConfigured(t *testing.T) {
	err := SendEmail(EmailConfig{}, "to@test.com", "subj", "body")
	if err == nil {
		t.Fatal("expected error for unconfigured email")
	}
}

func TestReadEmails_NotConfigured(t *testing.T) {
	_, err := ReadEmails(EmailConfig{}, 5)
	if err == nil {
		t.Fatal("expected error for unconfigured email")
	}
}

func TestEmailSummary_String(t *testing.T) {
	s := EmailSummary{From: "a@b.com", Subject: "Hello", Date: "2024-01-01"}
	str := s.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
}
