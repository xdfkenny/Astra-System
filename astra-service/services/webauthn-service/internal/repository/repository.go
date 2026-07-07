// Package repository implements persistence for WebAuthn credentials and
// verification sessions. It reads credential data from the employees and users
// tables and stores pending challenges in a lightweight sessions table.
package repository

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Common domain errors returned by the repository.
var (
	ErrCredentialNotFound = errors.New("repository: credential not found")
	ErrSessionNotFound    = errors.New("repository: session not found")
)

// ActorType identifies whether a credential belongs to an employee or admin user.
type ActorType string

const (
	ActorTypeEmployee ActorType = "employee"
	ActorTypeUser     ActorType = "user"
)

// Credential is a stored WebAuthn credential.
type Credential struct {
	ActorID            string
	ActorType          ActorType
	StoreID            string
	TenantID           string
	CredentialID       string
	PublicKey          []byte
	IsActive           bool
	LastLoginAt        *time.Time
	CreatedAt          time.Time
}

// Session stores a pending challenge for assertion verification.
type Session struct {
	SessionID      string
	ActorID        string
	ActorType      ActorType
	Challenge      string
	StoreID        string
	KioskID        string
	TenantID       string
	Reason         string
	RelyingPartyID string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// Repository is the persistence contract for WebAuthn credentials and sessions.
type Repository interface {
	GetCredential(ctx context.Context, actorID string, actorType ActorType) (*Credential, error)
	GetCredentialByCredentialID(ctx context.Context, credentialID string) (*Credential, error)
	SaveCredential(ctx context.Context, cred *Credential) error
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, actorID string, actorType ActorType, challenge string) (*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	TouchCredential(ctx context.Context, actorID string, actorType ActorType) error
}

// PostgresRepository is the production implementation backed by PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository returns a repository backed by the supplied *sql.DB.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// GetCredential loads a WebAuthn credential for an actor.
func (r *PostgresRepository) GetCredential(ctx context.Context, actorID string, actorType ActorType) (*Credential, error) {
	switch actorType {
	case ActorTypeEmployee:
		return r.queryEmployeeCredential(ctx, `SELECT employee_id, store_id, webauthn_credential_id, webauthn_public_key, is_active, last_login_at, created_at
			FROM employees WHERE employee_id = $1 AND deleted_at IS NULL`, actorID)
	case ActorTypeUser:
		return r.queryUserCredential(ctx, `SELECT user_id, tenant_id, webauthn_credential_id, webauthn_public_key, is_active, last_login_at, created_at
			FROM users WHERE user_id = $1 AND deleted_at IS NULL`, actorID)
	default:
		return nil, fmt.Errorf("repository: unsupported actor type %q", actorType)
	}
}

// GetCredentialByCredentialID loads a credential by its WebAuthn credential ID.
func (r *PostgresRepository) GetCredentialByCredentialID(ctx context.Context, credentialID string) (*Credential, error) {
	credBytes, err := base64.RawURLEncoding.DecodeString(credentialID)
	if err != nil {
		credBytes = []byte(credentialID)
	}

	var cred Credential
	if err := r.db.QueryRowContext(ctx, `
		SELECT employee_id, store_id, webauthn_credential_id, webauthn_public_key, is_active, last_login_at, created_at
		FROM employees WHERE webauthn_credential_id = $1 AND deleted_at IS NULL
		LIMIT 1`, base64.RawURLEncoding.EncodeToString(credBytes),
	).Scan(
		&cred.ActorID, &cred.StoreID, &cred.CredentialID, &cred.PublicKey, &cred.IsActive, &cred.LastLoginAt, &cred.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return r.getUserCredentialByCredentialID(ctx, credentialID)
		}
		return nil, fmt.Errorf("repository: query employee credential: %w", err)
	}
	cred.ActorType = ActorTypeEmployee
	cred.TenantID = ""
	return &cred, nil
}

func (r *PostgresRepository) getUserCredentialByCredentialID(ctx context.Context, credentialID string) (*Credential, error) {
	credBytes, err := base64.RawURLEncoding.DecodeString(credentialID)
	if err != nil {
		credBytes = []byte(credentialID)
	}

	var cred Credential
	if err := r.db.QueryRowContext(ctx, `
		SELECT user_id, tenant_id, webauthn_credential_id, webauthn_public_key, is_active, last_login_at, created_at
		FROM users WHERE webauthn_credential_id = $1 AND deleted_at IS NULL
		LIMIT 1`, base64.RawURLEncoding.EncodeToString(credBytes),
	).Scan(
		&cred.ActorID, &cred.TenantID, &cred.CredentialID, &cred.PublicKey, &cred.IsActive, &cred.LastLoginAt, &cred.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		return nil, fmt.Errorf("repository: query user credential: %w", err)
	}
	cred.ActorType = ActorTypeUser
	cred.StoreID = ""
	return &cred, nil
}

func (r *PostgresRepository) queryEmployeeCredential(ctx context.Context, query string, args ...any) (*Credential, error) {
	var cred Credential
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&cred.ActorID, &cred.StoreID, &cred.CredentialID, &cred.PublicKey, &cred.IsActive, &cred.LastLoginAt, &cred.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		return nil, fmt.Errorf("repository: query employee credential: %w", err)
	}
	cred.ActorType = ActorTypeEmployee
	return &cred, nil
}

func (r *PostgresRepository) queryUserCredential(ctx context.Context, query string, args ...any) (*Credential, error) {
	var cred Credential
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&cred.ActorID, &cred.TenantID, &cred.CredentialID, &cred.PublicKey, &cred.IsActive, &cred.LastLoginAt, &cred.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		return nil, fmt.Errorf("repository: query user credential: %w", err)
	}
	cred.ActorType = ActorTypeUser
	return &cred, nil
}

// SaveCredential persists a WebAuthn credential for an actor. It updates the
// existing employees or users row using the standard table pattern.
func (r *PostgresRepository) SaveCredential(ctx context.Context, cred *Credential) error {
	switch cred.ActorType {
	case ActorTypeEmployee:
		_, err := r.db.ExecContext(ctx, `
			UPDATE employees
			SET webauthn_credential_id = $1, webauthn_public_key = $2, updated_at = NOW()
			WHERE employee_id = $3 AND deleted_at IS NULL`,
			cred.CredentialID, cred.PublicKey, cred.ActorID,
		)
		if err != nil {
			return fmt.Errorf("repository: save employee credential: %w", err)
		}
	case ActorTypeUser:
		_, err := r.db.ExecContext(ctx, `
			UPDATE users
			SET webauthn_credential_id = $1, webauthn_public_key = $2, updated_at = NOW()
			WHERE user_id = $3 AND deleted_at IS NULL`,
			cred.CredentialID, cred.PublicKey, cred.ActorID,
		)
		if err != nil {
			return fmt.Errorf("repository: save user credential: %w", err)
		}
	default:
		return fmt.Errorf("repository: unsupported actor type %q", cred.ActorType)
	}
	return nil
}

// CreateSession persists a pending challenge session.
func (r *PostgresRepository) CreateSession(ctx context.Context, session *Session) error {
	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	storeID := nullUUID(session.StoreID)
	kioskID := nullUUID(session.KioskID)
	tenantID := nullUUID(session.TenantID)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO webauthn_sessions (session_id, actor_id, actor_type, challenge, store_id, kiosk_id, tenant_id, reason, relying_party_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		session.SessionID, session.ActorID, string(session.ActorType), session.Challenge,
		storeID, kioskID, tenantID, session.Reason, session.RelyingPartyID, session.ExpiresAt, session.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: insert session: %w", err)
	}
	return nil
}

func nullUUID(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// GetSession loads a pending challenge session.
func (r *PostgresRepository) GetSession(ctx context.Context, actorID string, actorType ActorType, challenge string) (*Session, error) {
	var session Session
	if err := r.db.QueryRowContext(ctx, `
		SELECT session_id, actor_id, actor_type, challenge, store_id, kiosk_id, tenant_id, reason, relying_party_id, expires_at, created_at
		FROM webauthn_sessions
		WHERE actor_id = $1 AND actor_type = $2 AND challenge = $3 AND expires_at > NOW()`,
		actorID, string(actorType), challenge,
	).Scan(
		&session.SessionID, &session.ActorID, &session.ActorType, &session.Challenge,
		&session.StoreID, &session.KioskID, &session.TenantID, &session.Reason, &session.RelyingPartyID, &session.ExpiresAt, &session.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("repository: query session: %w", err)
	}
	return &session, nil
}

// DeleteSession removes a challenge session after use.
func (r *PostgresRepository) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM webauthn_sessions WHERE session_id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("repository: delete session: %w", err)
	}
	return nil
}

// TouchCredential updates the last login timestamp for an actor.
func (r *PostgresRepository) TouchCredential(ctx context.Context, actorID string, actorType ActorType) error {
	switch actorType {
	case ActorTypeEmployee:
		_, err := r.db.ExecContext(ctx, `UPDATE employees SET last_login_at = NOW() WHERE employee_id = $1`, actorID)
		if err != nil {
			return fmt.Errorf("repository: touch employee: %w", err)
		}
	case ActorTypeUser:
		_, err := r.db.ExecContext(ctx, `UPDATE users SET last_login_at = NOW() WHERE user_id = $1`, actorID)
		if err != nil {
			return fmt.Errorf("repository: touch user: %w", err)
		}
	default:
		return fmt.Errorf("repository: unsupported actor type %q", actorType)
	}
	return nil
}

// Ensure PostgresRepository satisfies the Repository interface.
var _ Repository = (*PostgresRepository)(nil)
