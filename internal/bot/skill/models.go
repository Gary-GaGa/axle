package skill

import "fmt"

// ModelInfo holds metadata for each Copilot model.
type ModelInfo struct {
	ID         string
	Provider   string  // "anthropic" | "openai" | "google"
	Multiplier float64 // Premium request multiplier (0 = included/free on paid plans)
	Category   string  // "included" | "premium"
}

// Provider constants and display labels.
const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
	ProviderGoogle    = "google"
)

// ProviderLabel returns an emoji + name for the provider.
var ProviderLabel = map[string]string{
	ProviderAnthropic: "🟣 Anthropic (Claude)",
	ProviderOpenAI:    "🟢 OpenAI (GPT)",
	ProviderGoogle:    "🔵 Google (Gemini)",
}

// ProviderOrder defines display order for provider selection menu.
var ProviderOrder = []string{ProviderAnthropic, ProviderOpenAI, ProviderGoogle}

// AvailableModels lists all models supported by the local Copilot CLI,
// with their premium request multiplier.
// Source: GitHub docs + `copilot --help` (verified 2026-02-26).
var AvailableModels = []ModelInfo{
	// ── Included (free on paid plans) ──
	{ID: "gpt-5-mini", Provider: ProviderOpenAI, Multiplier: 0, Category: "included"},
	{ID: "gpt-4.1", Provider: ProviderOpenAI, Multiplier: 0, Category: "included"},
	// ── Premium models ──
	{ID: "claude-sonnet-4.6", Provider: ProviderAnthropic, Multiplier: 1, Category: "premium"},
	{ID: "claude-sonnet-4.5", Provider: ProviderAnthropic, Multiplier: 1, Category: "premium"},
	{ID: "claude-haiku-4.5", Provider: ProviderAnthropic, Multiplier: 0.25, Category: "premium"},
	{ID: "claude-sonnet-4", Provider: ProviderAnthropic, Multiplier: 1, Category: "premium"},
	{ID: "claude-opus-4.6", Provider: ProviderAnthropic, Multiplier: 3, Category: "premium"},
	{ID: "claude-opus-4.6-fast", Provider: ProviderAnthropic, Multiplier: 33, Category: "premium"},
	{ID: "claude-opus-4.5", Provider: ProviderAnthropic, Multiplier: 3, Category: "premium"},
	{ID: "gemini-3-pro-preview", Provider: ProviderGoogle, Multiplier: 1, Category: "premium"},
	{ID: "gpt-5.3-codex", Provider: ProviderOpenAI, Multiplier: 3, Category: "premium"},
	{ID: "gpt-5.2-codex", Provider: ProviderOpenAI, Multiplier: 3, Category: "premium"},
	{ID: "gpt-5.2", Provider: ProviderOpenAI, Multiplier: 1, Category: "premium"},
	{ID: "gpt-5.1-codex-max", Provider: ProviderOpenAI, Multiplier: 3, Category: "premium"},
	{ID: "gpt-5.1-codex", Provider: ProviderOpenAI, Multiplier: 1.5, Category: "premium"},
	{ID: "gpt-5.1", Provider: ProviderOpenAI, Multiplier: 1, Category: "premium"},
	{ID: "gpt-5.1-codex-mini", Provider: ProviderOpenAI, Multiplier: 0.5, Category: "premium"},
}

// ModelsByProvider returns models filtered by provider name.
func ModelsByProvider(provider string) []ModelInfo {
	var result []ModelInfo
	for _, m := range AvailableModels {
		if m.Provider == provider {
			result = append(result, m)
		}
	}
	return result
}

// ModelLabel returns a display label with cost indicator for the model menu.
func (m ModelInfo) ModelLabel() string {
	if m.Multiplier == 0 {
		return fmt.Sprintf("✅ %s [免費]", m.ID)
	}
	return fmt.Sprintf("%s [%gx]", m.ID, m.Multiplier)
}

// DefaultModel is used when the user hasn't selected a model.
const DefaultModel = "claude-sonnet-4.6"

// MaxPromptChars is the maximum prompt length sent to Copilot CLI.
// Longer prompts are truncated to avoid hitting context limits.
const MaxPromptChars = 8000

// MaxResponseChars is the safe per-message character limit for Telegram.
const MaxResponseChars = 4000

// SplitMessage splits text into chunks ≤ MaxResponseChars, splitting
// preferably at newline boundaries.
func SplitMessage(text string) []string {
	if len(text) <= MaxResponseChars {
		return []string{text}
	}
	var chunks []string
	for len(text) > MaxResponseChars {
		cutAt := MaxResponseChars
		if idx := lastNewlineBefore(text, cutAt); idx > MaxResponseChars/2 {
			cutAt = idx
		}
		chunks = append(chunks, text[:cutAt])
		text = text[cutAt:]
	}
	if text != "" {
		chunks = append(chunks, text)
	}
	return chunks
}

func lastNewlineBefore(s string, pos int) int {
	for i := pos - 1; i >= 0; i-- {
		if s[i] == '\n' {
			return i + 1
		}
	}
	return -1
}
