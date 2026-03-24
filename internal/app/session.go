package app

import interaction "github.com/garyellow/axle/internal/domain/interaction"

// Mode describes what input the bot is currently expecting from a user.
type Mode = interaction.Mode

const (
	ModeIdle                 = interaction.ModeIdle
	ModeAwaitReadPath        = interaction.ModeAwaitReadPath
	ModeAwaitExecCmd         = interaction.ModeAwaitExecCmd
	ModeAwaitExecConfirm     = interaction.ModeAwaitExecConfirm
	ModeAwaitCopilotPrompt   = interaction.ModeAwaitCopilotPrompt
	ModeAwaitWritePath       = interaction.ModeAwaitWritePath
	ModeAwaitWriteContent    = interaction.ModeAwaitWriteContent
	ModeAwaitWebSearch       = interaction.ModeAwaitWebSearch
	ModeAwaitWebURL          = interaction.ModeAwaitWebURL
	ModeAwaitMemorySearch    = interaction.ModeAwaitMemorySearch
	ModeAwaitBrowserScript   = interaction.ModeAwaitBrowserScript
	ModeAwaitWorkflowRequest = interaction.ModeAwaitWorkflowRequest
	ModeAwaitProjectPath     = interaction.ModeAwaitProjectPath
	ModeAwaitListPath        = interaction.ModeAwaitListPath
	ModeAwaitSearchQuery     = interaction.ModeAwaitSearchQuery
	ModeAwaitGitCommitMsg    = interaction.ModeAwaitGitCommitMsg
	ModeAwaitSubAgentName    = interaction.ModeAwaitSubAgentName
	ModeAwaitSubAgentTask    = interaction.ModeAwaitSubAgentTask
	ModeAwaitSchedName       = interaction.ModeAwaitSchedName
	ModeAwaitSchedInterval   = interaction.ModeAwaitSchedInterval
	ModeAwaitSchedCommand    = interaction.ModeAwaitSchedCommand
	ModeAwaitEmailTo         = interaction.ModeAwaitEmailTo
	ModeAwaitEmailSubject    = interaction.ModeAwaitEmailSubject
	ModeAwaitEmailBody       = interaction.ModeAwaitEmailBody
	ModeAwaitGHPRTitle       = interaction.ModeAwaitGHPRTitle
	ModeAwaitGHPRBody        = interaction.ModeAwaitGHPRBody
	ModeAwaitUpgradeRequest  = interaction.ModeAwaitUpgradeRequest
)

// UserSession holds per-user interaction state between messages.
type UserSession = interaction.UserSession

// SessionManager manages per-user sessions. Goroutine-safe.
type SessionManager = interaction.SessionManager

// NewSessionManager creates a ready-to-use SessionManager.
func NewSessionManager() *SessionManager {
	return interaction.NewSessionManager()
}
