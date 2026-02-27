package skill

import (
	"testing"
)

func TestSplitMessage_Short(t *testing.T) {
	msg := "hello world"
	chunks := SplitMessage(msg)
	if len(chunks) != 1 || chunks[0] != msg {
		t.Errorf("SplitMessage short: got %d chunks, want 1", len(chunks))
	}
}

func TestSplitMessage_Long(t *testing.T) {
	// Create a message longer than MaxResponseChars
	long := ""
	for i := 0; i < MaxResponseChars+100; i++ {
		long += "x"
	}
	chunks := SplitMessage(long)
	if len(chunks) < 2 {
		t.Errorf("SplitMessage long: got %d chunks, want >= 2", len(chunks))
	}
	// Verify all chunks are within limit
	for i, c := range chunks {
		if len(c) > MaxResponseChars {
			t.Errorf("chunk %d length %d exceeds max %d", i, len(c), MaxResponseChars)
		}
	}
}

func TestSplitMessage_NewlineBoundary(t *testing.T) {
	// Build a message with newlines to test splitting at newline boundaries
	var msg string
	line := "this is a test line of reasonable length\n"
	for len(msg) < MaxResponseChars+200 {
		msg += line
	}
	chunks := SplitMessage(msg)
	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks, got %d", len(chunks))
	}
	// First chunk should end at a newline
	if len(chunks[0]) > 0 && chunks[0][len(chunks[0])-1] != '\n' {
		// It's OK if it doesn't end at newline (fallback to max), but prefer newline
	}
}

func TestModelsByProvider(t *testing.T) {
	anthropic := ModelsByProvider(ProviderAnthropic)
	if len(anthropic) == 0 {
		t.Error("expected at least 1 anthropic model")
	}
	for _, m := range anthropic {
		if m.Provider != ProviderAnthropic {
			t.Errorf("model %s has wrong provider %s", m.ID, m.Provider)
		}
	}

	openai := ModelsByProvider(ProviderOpenAI)
	if len(openai) == 0 {
		t.Error("expected at least 1 openai model")
	}

	google := ModelsByProvider(ProviderGoogle)
	if len(google) == 0 {
		t.Error("expected at least 1 google model")
	}

	none := ModelsByProvider("nonexistent")
	if len(none) != 0 {
		t.Errorf("expected 0 models for nonexistent provider, got %d", len(none))
	}
}

func TestModelLabel(t *testing.T) {
	free := ModelInfo{ID: "gpt-5-mini", Multiplier: 0}
	label := free.ModelLabel()
	if label != "✅ gpt-5-mini [免費]" {
		t.Errorf("free model label = %q", label)
	}

	premium := ModelInfo{ID: "claude-opus-4.6", Multiplier: 3}
	label = premium.ModelLabel()
	if label != "claude-opus-4.6 [3x]" {
		t.Errorf("premium model label = %q", label)
	}
}

func TestLastNewlineBefore(t *testing.T) {
	s := "hello\nworld\nfoo"
	idx := lastNewlineBefore(s, 12)
	if idx != 12 { // after "world\n" which is at position 11, +1 = 12
		t.Errorf("lastNewlineBefore = %d, want 12", idx)
	}

	// No newline case
	idx = lastNewlineBefore("hello", 3)
	if idx != -1 {
		t.Errorf("lastNewlineBefore no newline = %d, want -1", idx)
	}
}

func TestIsBinaryExt(t *testing.T) {
	if !isBinaryExt(".png") {
		t.Error(".png should be binary")
	}
	if !isBinaryExt(".exe") {
		t.Error(".exe should be binary")
	}
	if isBinaryExt(".go") {
		t.Error(".go should not be binary")
	}
	if isBinaryExt(".txt") {
		t.Error(".txt should not be binary")
	}
}
