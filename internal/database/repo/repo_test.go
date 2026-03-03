package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JinFuuMugen/GophKeeper/internal/database/repo"
	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func newMockRepo(t *testing.T) (*repo.Repo, pgxmock.PgxPoolIface) {
	t.Helper()

	pool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}

	r := repo.NewRepoFromPool(pool)
	return r, pool
}

func TestNewRepoFromPool(t *testing.T) {
	_, pool := newMockRepo(t)
	defer pool.Close()
}

func TestCreateUser_OK(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	id := uuid.New()

	pool.ExpectExec(`INSERT INTO users \(id, login, password_hash\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs(id, "alice", "hash").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err := r.CreateUser(context.Background(), id, "alice", "hash")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestCreateUser_Duplicate_ReturnsErrUserExists(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	id := uuid.New()

	pool.ExpectExec(`INSERT INTO users \(id, login, password_hash\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs(id, "alice", "hash").
		WillReturnError(&pgconn.PgError{Code: "23505"})

	err := r.CreateUser(context.Background(), id, "alice", "hash")
	if !errors.Is(err, errdefs.ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got %v", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestCreateUser_OtherError_Wrapped(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	id := uuid.New()

	pool.ExpectExec(`INSERT INTO users \(id, login, password_hash\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs(id, "alice", "hash").
		WillReturnError(errors.New("db"))

	err := r.CreateUser(context.Background(), id, "alice", "hash")
	if err == nil {
		t.Fatalf("expected error")
	}
	if errors.Is(err, errdefs.ErrUserExists) {
		t.Fatalf("did not expect ErrUserExists")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetUserByLogin_OK(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	id := uuid.New()
	now := time.Unix(0, 0).UTC()
	salt := []byte{1, 2, 3}

	rows := pgxmock.NewRows([]string{"id", "login", "password_hash", "created_at", "kdf_salt"}).
		AddRow(id, "alice", "hash", now, salt)

	pool.ExpectQuery(`SELECT id, login, password_hash, created_at, kdf_salt FROM users WHERE login=\$1`).
		WithArgs("alice").
		WillReturnRows(rows)

	u, err := r.GetUserByLogin(context.Background(), "alice")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if u.ID != id || u.Login != "alice" || u.PasswordHash != "hash" || !u.CreatedAt.Equal(now) {
		t.Fatalf("unexpected user: %+v", u)
	}
	if string(u.KDFSalt) != string(salt) {
		t.Fatalf("unexpected salt: %v", u.KDFSalt)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetUserByLogin_NotFound(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	pool.ExpectQuery(`SELECT id, login, password_hash, created_at, kdf_salt FROM users WHERE login=\$1`).
		WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	_, err := r.GetUserByLogin(context.Background(), "missing")
	if !errors.Is(err, errdefs.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetUserByLogin_OtherError_Wrapped(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	pool.ExpectQuery(`SELECT id, login, password_hash, created_at, kdf_salt FROM users WHERE login=\$1`).
		WithArgs("alice").
		WillReturnError(errors.New("db"))

	_, err := r.GetUserByLogin(context.Background(), "alice")
	if err == nil {
		t.Fatalf("expected error")
	}
	if errors.Is(err, errdefs.ErrUserNotFound) {
		t.Fatalf("did not expect ErrUserNotFound")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExistsLogin_True(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	rows := pgxmock.NewRows([]string{"?column?"}).AddRow(1)

	pool.ExpectQuery(`SELECT 1 FROM users WHERE login=\$1`).
		WithArgs("alice").
		WillReturnRows(rows)

	ok, err := r.ExistsLogin(context.Background(), "alice")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if !ok {
		t.Fatalf("expected true")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExistsLogin_False(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	pool.ExpectQuery(`SELECT 1 FROM users WHERE login=\$1`).
		WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	ok, err := r.ExistsLogin(context.Background(), "missing")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if ok {
		t.Fatalf("expected false")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestExistsLogin_Error(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	pool.ExpectQuery(`SELECT 1 FROM users WHERE login=\$1`).
		WithArgs("alice").
		WillReturnError(errors.New("db"))

	ok, err := r.ExistsLogin(context.Background(), "alice")
	if err == nil {
		t.Fatalf("expected error")
	}
	if ok {
		t.Fatalf("expected false")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpsertItem_Success_ReturnsRow(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	now := time.Unix(0, 0).UTC()

	in := models.Item{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      "text",
		Encrypted: []byte("enc"),
		Metadata:  "m",
		Version:   2,
		Deleted:   false,
	}

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "type", "encrypted", "metadata", "version", "deleted", "created_at", "updated_at",
	}).AddRow(in.ID, in.UserID, in.Type, in.Encrypted, in.Metadata, in.Version, in.Deleted, now, now)

	pool.ExpectQuery(`INSERT INTO items .* RETURNING id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at`).
		WithArgs(in.ID, in.UserID, in.Type, in.Encrypted, in.Metadata, in.Version, in.Deleted).
		WillReturnRows(rows)

	out, err := r.UpsertItem(context.Background(), in)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if out.ID != in.ID || out.UserID != in.UserID || out.Type != in.Type || out.Metadata != in.Metadata || out.Version != in.Version || out.Deleted != in.Deleted {
		t.Fatalf("unexpected out: %+v", out)
	}
	if string(out.Encrypted) != string(in.Encrypted) {
		t.Fatalf("unexpected encrypted: %v", out.Encrypted)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestUpsertItem_FallbackToGetItem(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	now := time.Unix(0, 0).UTC()

	in := models.Item{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      "text",
		Encrypted: []byte("enc"),
		Metadata:  "m",
		Version:   1,
		Deleted:   false,
	}

	pool.ExpectQuery(`INSERT INTO items .* RETURNING id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at`).
		WithArgs(in.ID, in.UserID, in.Type, in.Encrypted, in.Metadata, in.Version, in.Deleted).
		WillReturnError(pgx.ErrNoRows)

	outRows := pgxmock.NewRows([]string{
		"id", "user_id", "type", "encrypted", "metadata", "version", "deleted", "created_at", "updated_at",
	}).AddRow(in.ID, in.UserID, "text", []byte("old"), "old", int64(1), false, now, now)

	pool.ExpectQuery(`SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at FROM items WHERE user_id=\$1 AND id=\$2`).
		WithArgs(in.UserID, in.ID).
		WillReturnRows(outRows)

	out, err := r.UpsertItem(context.Background(), in)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if out.Metadata != "old" || out.Version != 1 {
		t.Fatalf("unexpected out: %+v", out)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetItem_OK(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	now := time.Unix(0, 0).UTC()
	uid := uuid.New()
	id := uuid.New()

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "type", "encrypted", "metadata", "version", "deleted", "created_at", "updated_at",
	}).AddRow(id, uid, "text", []byte("enc"), "m", int64(1), false, now, now)

	pool.ExpectQuery(`SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at FROM items WHERE user_id=\$1 AND id=\$2`).
		WithArgs(uid, id).
		WillReturnRows(rows)

	out, err := r.GetItem(context.Background(), uid, id)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if out.ID != id || out.UserID != uid || out.Metadata != "m" || out.Version != 1 {
		t.Fatalf("unexpected out: %+v", out)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetItem_Error_Wrapped(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	uid := uuid.New()
	id := uuid.New()

	pool.ExpectQuery(`SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at FROM items WHERE user_id=\$1 AND id=\$2`).
		WithArgs(uid, id).
		WillReturnError(errors.New("db"))

	_, err := r.GetItem(context.Background(), uid, id)
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListItems_OK(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	uid := uuid.New()
	now := time.Unix(0, 0).UTC()

	id1 := uuid.New()
	id2 := uuid.New()

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "type", "encrypted", "metadata", "version", "deleted", "created_at", "updated_at",
	}).
		AddRow(id1, uid, "text", []byte("a"), "m1", int64(1), false, now, now).
		AddRow(id2, uid, "text", []byte("b"), "m2", int64(2), true, now, now)

	pool.ExpectQuery(`SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at FROM items WHERE user_id=\$1 ORDER BY updated_at ASC`).
		WithArgs(uid).
		WillReturnRows(rows)

	out, err := r.ListItems(context.Background(), uid)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2, got %d", len(out))
	}
	if out[0].ID != id1 || out[1].ID != id2 {
		t.Fatalf("unexpected ids: %+v", out)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListItems_QueryError(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	uid := uuid.New()

	pool.ExpectQuery(`SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at FROM items WHERE user_id=\$1 ORDER BY updated_at ASC`).
		WithArgs(uid).
		WillReturnError(errors.New("db"))

	_, err := r.ListItems(context.Background(), uid)
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListItems_ScanError(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	uid := uuid.New()
	now := time.Unix(0, 0).UTC()

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "type", "encrypted", "metadata", "version", "deleted", "created_at", "updated_at",
	}).AddRow("not-a-uuid", uid, "text", []byte("a"), "m1", int64(1), false, now, now)

	pool.ExpectQuery(`SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at FROM items WHERE user_id=\$1 ORDER BY updated_at ASC`).
		WithArgs(uid).
		WillReturnRows(rows)

	_, err := r.ListItems(context.Background(), uid)
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListItemsSince_OK(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	uid := uuid.New()
	now := time.Unix(0, 0).UTC()
	since := now.Add(-time.Hour)

	id1 := uuid.New()

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "type", "encrypted", "metadata", "version", "deleted", "created_at", "updated_at",
	}).AddRow(id1, uid, "text", []byte("a"), "m1", int64(1), false, now, now)

	pool.ExpectQuery(`SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at FROM items WHERE user_id=\$1 AND updated_at > \$2 ORDER BY updated_at ASC`).
		WithArgs(uid, since).
		WillReturnRows(rows)

	out, err := r.ListItemsSince(context.Background(), uid, since)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(out) != 1 || out[0].ID != id1 {
		t.Fatalf("unexpected out: %+v", out)
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListItemsSince_QueryError(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	uid := uuid.New()
	since := time.Unix(0, 0).UTC()

	pool.ExpectQuery(`SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at FROM items WHERE user_id=\$1 AND updated_at > \$2 ORDER BY updated_at ASC`).
		WithArgs(uid, since).
		WillReturnError(errors.New("db"))

	_, err := r.ListItemsSince(context.Background(), uid, since)
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestListItemsSince_ScanError(t *testing.T) {
	r, pool := newMockRepo(t)
	defer pool.Close()

	uid := uuid.New()
	now := time.Unix(0, 0).UTC()
	since := now.Add(-time.Hour)

	rows := pgxmock.NewRows([]string{
		"id", "user_id", "type", "encrypted", "metadata", "version", "deleted", "created_at", "updated_at",
	}).AddRow("not-a-uuid", uid, "text", []byte("a"), "m1", int64(1), false, now, now)

	pool.ExpectQuery(`SELECT id, user_id, type, encrypted, metadata, version, deleted, created_at, updated_at FROM items WHERE user_id=\$1 AND updated_at > \$2 ORDER BY updated_at ASC`).
		WithArgs(uid, since).
		WillReturnRows(rows)

	_, err := r.ListItemsSince(context.Background(), uid, since)
	if err == nil {
		t.Fatalf("expected error")
	}

	if err := pool.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
