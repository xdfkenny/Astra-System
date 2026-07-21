package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	StepWSL    = "wsl"
	StepDocker = "docker"
	StepDeploy = "deploy"
	StepDone   = "done"
)

type State struct {
	Step      string    `json:"step"`
	Version   string    `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
	path      string
}

func Load(dataDir string) (*State, error) {
	s := &State{
		Step:    StepWSL,
		Version: "0.2.0",
		path:    filepath.Join(dataDir, "state.json"),
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}

	return s, nil
}

func (s *State) Save() error {
	s.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		dir := filepath.Dir(s.path)
		if os.MkdirAll(dir, 0755) == nil {
			err = os.WriteFile(s.path, data, 0644)
		}
		if err != nil {
			return fmt.Errorf("write state: %w", err)
		}
	}
	return nil
}
