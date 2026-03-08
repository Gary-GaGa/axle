package app

import "sync"

// Mode describes what input the bot is currently expecting from a user.
type Mode int

const (
	ModeIdle                 Mode = iota
	ModeAwaitReadPath             // Next text message = file path to read
	ModeAwaitExecCmd              // Next text message = shell command to execute
	ModeAwaitExecConfirm          // Waiting for ✅/❌ button press (block new inputs)
	ModeAwaitCopilotPrompt        // Next text message = prompt for Copilot CLI
	ModeAwaitWritePath            // Next text message = file path to write
	ModeAwaitWriteContent         // Next text message = content to write into file
	ModeAwaitWebSearch            // Next text message = search query
	ModeAwaitWebURL               // Next text message = URL to fetch
	ModeAwaitMemorySearch         // Next text message = memory/history query
	ModeAwaitBrowserScript        // Next text message = browser automation script
	ModeAwaitWorkflowRequest      // Next text message = workflow request
	ModeAwaitProjectPath          // Next text message = absolute path for workspace switch
	ModeAwaitListPath             // Next text message = directory path to list
	ModeAwaitSearchQuery          // Next text message = code search keyword
	ModeAwaitGitCommitMsg         // Next text message = git commit message
	ModeAwaitSubAgentName         // Next text message = sub-agent name
	ModeAwaitSubAgentTask         // Next text message = sub-agent task description
	ModeAwaitSchedName            // Next text message = schedule name
	ModeAwaitSchedInterval        // Next text message = schedule interval in minutes
	ModeAwaitSchedCommand         // Next text message = schedule command
	ModeAwaitEmailTo              // Next text message = email recipient
	ModeAwaitEmailSubject         // Next text message = email subject
	ModeAwaitEmailBody            // Next text message = email body
	ModeAwaitGHPRTitle            // Next text message = PR title
	ModeAwaitGHPRBody             // Next text message = PR body
	ModeAwaitUpgradeRequest       // Next text message = feature request for self-upgrade
)

// UserSession holds per-user interaction state between messages.
type UserSession struct {
	Mode               Mode
	PendingCmd         string          // Shell command awaiting confirmation
	PendingPath        string          // File path awaiting content (write flow)
	SelectedModel      string          // Chosen Copilot model
	ActiveWorkspace    string          // User-selected workspace (empty = use default)
	PendingAgent       string          // Sub-agent name being created
	PendingSchedName   string          // Schedule name being created
	PendingSchedCmd    string          // Schedule command being created
	PendingEmailTo     string          // Email recipient being composed
	PendingEmailSubj   string          // Email subject being composed
	PendingPRTitle     string          // PR title being created
	PendingUpgradeReq  string          // Feature request for self-upgrade
	PendingUpgradePlan string          // Copilot-generated upgrade plan
	EnabledExtras      map[string]bool // Optional features pinned to main menu
}

// SessionManager manages per-user sessions. Goroutine-safe.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[int64]*UserSession
}

// NewSessionManager creates a ready-to-use SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[int64]*UserSession)}
}

// GetCopy returns a value copy of the user's session (safe for concurrent reads).
func (sm *SessionManager) GetCopy(userID int64) UserSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.sessions[userID]
	if !ok {
		return UserSession{}
	}
	cp := *s
	cp.EnabledExtras = cloneExtras(s.EnabledExtras)
	return cp
}

// Update atomically reads and modifies a user's session under a write lock.
// The fn callback receives a pointer to the live session struct.
func (sm *SessionManager) Update(userID int64, fn func(*UserSession)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[userID]
	if !ok {
		s = &UserSession{}
		sm.sessions[userID] = s
	}
	fn(s)
}

// Reset clears a user's session back to idle state, preserving SelectedModel and ActiveWorkspace.
func (sm *SessionManager) Reset(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	s, ok := sm.sessions[userID]
	if !ok {
		sm.sessions[userID] = &UserSession{}
		return
	}
	model := s.SelectedModel
	ws := s.ActiveWorkspace
	extras := cloneExtras(s.EnabledExtras)
	*s = UserSession{SelectedModel: model, ActiveWorkspace: ws, EnabledExtras: extras}
}

func cloneExtras(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
