package tools

import (
	artifactspkg "github.com/channyeintun/nami/internal/artifacts"
	"github.com/channyeintun/nami/internal/session"
)

type InteractionRuntimeConfig struct {
	BackgroundCommandNotifier func(BackgroundCommandUpdate)
	AskUserQuestionRuntime    AskUserQuestionRuntime
	SessionControlRuntime     SessionControlRuntime
	ToolSearchRuntime         ToolSearchRuntime
}

func InstallInteractionRuntime(cfg InteractionRuntimeConfig) {
	SetBackgroundCommandNotifier(cfg.BackgroundCommandNotifier)
	SetAskUserQuestionRuntime(cfg.AskUserQuestionRuntime)
	SetSessionControlRuntime(cfg.SessionControlRuntime)
	SetToolSearchRuntime(cfg.ToolSearchRuntime)
}

func ClearInteractionRuntime() {
	InstallInteractionRuntime(InteractionRuntimeConfig{})
}

type SessionArtifactRuntimeConfig struct {
	SessionID string
	Manager   *artifactspkg.Manager
}

type SwarmRuntimeConfig struct {
	SessionID string
	Manager   *artifactspkg.Manager
	Store     *session.Store
	CWD       string
}

type SessionRuntimeConfig struct {
	FileHistory      *FileHistory
	FileReadState    *FileReadState
	SessionArtifacts SessionArtifactRuntimeConfig
	Swarm            SwarmRuntimeConfig
}

func InstallSessionRuntime(cfg SessionRuntimeConfig) {
	SetGlobalFileHistory(cfg.FileHistory)
	SetGlobalFileReadState(cfg.FileReadState)
	SetGlobalSessionArtifacts(cfg.SessionArtifacts.SessionID, cfg.SessionArtifacts.Manager)
	SetGlobalSwarmRuntime(cfg.Swarm.SessionID, cfg.Swarm.Manager, cfg.Swarm.Store, cfg.Swarm.CWD)
}

func ClearSessionRuntime() {
	InstallSessionRuntime(SessionRuntimeConfig{})
}
