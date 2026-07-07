package repository

import (
	"context"
	"sync"
	"time"
)

// MemoryRepository is an in-memory implementation of Repository for unit tests.
type MemoryRepository struct {
	mu          sync.RWMutex
	credentials map[string]*Credential
	sessions    map[string]*Session
}

// NewMemoryRepository returns an empty in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		credentials: make(map[string]*Credential),
		sessions:    make(map[string]*Session),
	}
}

func key(actorID string, actorType ActorType) string {
	return actorID + "|" + string(actorType)
}

// GetCredential loads a credential by actor.
func (r *MemoryRepository) GetCredential(ctx context.Context, actorID string, actorType ActorType) (*Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cred, ok := r.credentials[key(actorID, actorType)]
	if !ok {
		return nil, ErrCredentialNotFound
	}
	return cred, nil
}

// GetCredentialByCredentialID loads a credential by credential ID.
func (r *MemoryRepository) GetCredentialByCredentialID(ctx context.Context, credentialID string) (*Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, cred := range r.credentials {
		if cred.CredentialID == credentialID {
			return cred, nil
		}
	}
	return nil, ErrCredentialNotFound
}

// SaveCredential persists a credential.
func (r *MemoryRepository) SaveCredential(ctx context.Context, cred *Credential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.credentials[key(cred.ActorID, cred.ActorType)] = cred
	return nil
}

// CreateSession persists a session.
func (r *MemoryRepository) CreateSession(ctx context.Context, session *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if session.SessionID == "" {
		session.SessionID = "session-" + key(session.ActorID, session.ActorType) + "-" + session.Challenge
	}
	session.CreatedAt = time.Now().UTC()
	r.sessions[key(session.ActorID, session.ActorType)+"|"+session.Challenge] = session
	return nil
}

// GetSession loads a session by actor and challenge.
func (r *MemoryRepository) GetSession(ctx context.Context, actorID string, actorType ActorType, challenge string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[key(actorID, actorType)+"|"+challenge]
	if !ok || session.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// DeleteSession removes a session.
func (r *MemoryRepository) DeleteSession(ctx context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, s := range r.sessions {
		if s.SessionID == sessionID {
			delete(r.sessions, k)
			return nil
		}
	}
	return nil
}

// TouchCredential updates the last login timestamp.
func (r *MemoryRepository) TouchCredential(ctx context.Context, actorID string, actorType ActorType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cred, ok := r.credentials[key(actorID, actorType)]
	if !ok {
		return ErrCredentialNotFound
	}
	now := time.Now().UTC()
	cred.LastLoginAt = &now
	return nil
}

// SetCredential is a test helper to seed a credential.
func (r *MemoryRepository) SetCredential(cred *Credential) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.credentials[key(cred.ActorID, cred.ActorType)] = cred
}

// Ensure MemoryRepository satisfies Repository.
var _ Repository = (*MemoryRepository)(nil)
