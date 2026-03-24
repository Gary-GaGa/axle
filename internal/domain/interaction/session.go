package interaction

import "sync"

// Mode describes what input the assistant is currently expecting from a user.
type Mode int

const (
	ModeIdle Mode = iota
	ModeAwaitReadPath
	ModeAwaitExecCmd
	ModeAwaitExecConfirm
	ModeAwaitCopilotPrompt
	ModeAwaitWritePath
	ModeAwaitWriteContent
	ModeAwaitWebSearch
	ModeAwaitWebURL
	ModeAwaitMemorySearch
	ModeAwaitBrowserScript
	ModeAwaitWorkflowRequest
	ModeAwaitProjectPath
	ModeAwaitListPath
	ModeAwaitSearchQuery
	ModeAwaitGitCommitMsg
	ModeAwaitSubAgentName
	ModeAwaitSubAgentTask
	ModeAwaitSchedName
	ModeAwaitSchedInterval
	ModeAwaitSchedCommand
	ModeAwaitEmailTo
	ModeAwaitEmailSubject
	ModeAwaitEmailBody
	ModeAwaitGHPRTitle
	ModeAwaitGHPRBody
	ModeAwaitUpgradeRequest
)

// UserSession holds per-user interaction state between messages.
type UserSession struct {
	Mode               Mode
	PendingCmd         string
	PendingPath        string
	SelectedModel      string
	ActiveWorkspace    string
	PendingAgent       string
	PendingSchedName   string
	PendingSchedCmd    string
	PendingEmailTo     string
	PendingEmailSubj   string
	PendingPRTitle     string
	PendingUpgradeReq  string
	PendingUpgradePlan string
	EnabledExtras      map[string]bool
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

// GetCopy returns a value copy of the user's session.
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

// Reset clears a user's session back to idle state while preserving stable preferences.
func (sm *SessionManager) Reset(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s, ok := sm.sessions[userID]
	if !ok {
		sm.sessions[userID] = &UserSession{}
		return
	}
	model := s.SelectedModel
	workspace := s.ActiveWorkspace
	extras := cloneExtras(s.EnabledExtras)
	*s = UserSession{SelectedModel: model, ActiveWorkspace: workspace, EnabledExtras: extras}
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
