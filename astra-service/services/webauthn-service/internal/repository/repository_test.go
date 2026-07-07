package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/astra-systems/astra-service/services/webauthn-service/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepositoryCredential(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	storeID := uuid.New().String()
	createStore(t, db, ctx, storeID)

	employeeID := uuid.New().String()
	credID := "cred-" + uuid.New().String()

	_, err := db.ExecContext(ctx, `
		INSERT INTO employees (employee_id, store_id, name, email, role, webauthn_credential_id, webauthn_public_key, is_active)
		VALUES ($1, $2, 'Test Employee', 'test@example.com', 'cashier', $3, $4, TRUE)`,
		employeeID, storeID, credID, []byte("public-key"),
	)
	require.NoError(t, err)

	cred, err := repo.GetCredential(ctx, employeeID, ActorTypeEmployee)
	require.NoError(t, err)
	assert.Equal(t, employeeID, cred.ActorID)
	assert.Equal(t, credID, cred.CredentialID)
	assert.True(t, cred.IsActive)

	_, err = repo.GetCredential(ctx, uuid.New().String(), ActorTypeEmployee)
	assert.ErrorIs(t, err, ErrCredentialNotFound)
}

func createStore(t *testing.T, db *sql.DB, ctx context.Context, storeID string) {
	t.Helper()
	_, err := db.ExecContext(ctx, `INSERT INTO stores (store_id, name) VALUES ($1, 'Test Store')`, storeID)
	require.NoError(t, err)
}

func TestPostgresRepositorySession(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	session := &Session{
		SessionID:      uuid.New().String(),
		ActorID:        uuid.New().String(),
		ActorType:      ActorTypeEmployee,
		Challenge:      "challenge-123",
		StoreID:        uuid.New().String(),
		KioskID:        uuid.New().String(),
		TenantID:       uuid.New().String(),
		Reason:         "override",
		RelyingPartyID: "astra.example.com",
		ExpiresAt:      time.Now().UTC().Add(time.Minute),
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	loaded, err := repo.GetSession(ctx, session.ActorID, ActorTypeEmployee, session.Challenge)
	require.NoError(t, err)
	assert.Equal(t, session.Challenge, loaded.Challenge)
	assert.Equal(t, session.RelyingPartyID, loaded.RelyingPartyID)

	require.NoError(t, repo.DeleteSession(ctx, session.SessionID))
	_, err = repo.GetSession(ctx, session.ActorID, ActorTypeEmployee, session.Challenge)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestPostgresRepositorySessionExpired(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	session := &Session{
		SessionID:      uuid.New().String(),
		ActorID:        uuid.New().String(),
		ActorType:      ActorTypeEmployee,
		Challenge:      "challenge-123",
		ExpiresAt:      time.Now().UTC().Add(-time.Minute),
		RelyingPartyID: "astra.example.com",
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	_, err := repo.GetSession(ctx, session.ActorID, ActorTypeEmployee, session.Challenge)
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestPostgresRepositorySaveCredential(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	employeeID := uuid.New().String()
	storeID := uuid.New().String()
	createStore(t, db, ctx, storeID)
	_, err := db.ExecContext(ctx, `
		INSERT INTO employees (employee_id, store_id, name, email, role, is_active)
		VALUES ($1, $2, 'Test Employee', 'save@example.com', 'cashier', TRUE)`,
		employeeID, storeID,
	)
	require.NoError(t, err)

	require.NoError(t, repo.SaveCredential(ctx, &Credential{
		ActorID:      employeeID,
		ActorType:    ActorTypeEmployee,
		CredentialID: "cred-save",
		PublicKey:    []byte("public-key"),
		IsActive:     true,
	}))

	cred, err := repo.GetCredential(ctx, employeeID, ActorTypeEmployee)
	require.NoError(t, err)
	assert.Equal(t, "cred-save", cred.CredentialID)
	assert.Equal(t, []byte("public-key"), cred.PublicKey)
}

func TestPostgresRepositoryTouchCredential(t *testing.T) {
	db, cleanup := testutil.NewPostgresContainer(t)
	defer cleanup()

	repo := NewPostgresRepository(db)
	ctx := context.Background()

	employeeID := uuid.New().String()
	storeID := uuid.New().String()
	createStore(t, db, ctx, storeID)
	_, err := db.ExecContext(ctx, `
		INSERT INTO employees (employee_id, store_id, name, email, role, webauthn_credential_id, webauthn_public_key, is_active)
		VALUES ($1, $2, 'Test Employee', 'touch@example.com', 'cashier', $3, $4, TRUE)`,
		employeeID, storeID, "cred-touch", []byte("public-key"),
	)
	require.NoError(t, err)

	require.NoError(t, repo.TouchCredential(ctx, employeeID, ActorTypeEmployee))

	cred, err := repo.GetCredential(ctx, employeeID, ActorTypeEmployee)
	require.NoError(t, err)
	assert.NotNil(t, cred.LastLoginAt)
}
