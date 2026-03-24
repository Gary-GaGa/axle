package workspace

import (
	"strings"
	"testing"
)

func TestValidateWritePath(t *testing.T) {
	if err := ValidateWritePath("notes/todo.txt"); err != nil {
		t.Fatalf("expected valid path, got %v", err)
	}
	if err := ValidateWritePath("   "); err == nil {
		t.Fatalf("expected blank path to be rejected")
	}
	if err := ValidateWritePath("../escape.txt"); err == nil {
		t.Fatalf("expected traversal path to be rejected")
	}
}

func TestValidateWriteContent(t *testing.T) {
	if err := ValidateWriteContent("hello"); err != nil {
		t.Fatalf("expected small content to pass, got %v", err)
	}
	err := ValidateWriteContent(strings.Repeat("a", MaxWriteBytes+1))
	if err == nil {
		t.Fatalf("expected oversized content to be rejected")
	}
}
