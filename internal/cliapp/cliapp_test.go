package cliapp_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JinFuuMugen/GophKeeper/internal/cliapp"
)

func TestCLI_TextLifecycle_EndToEnd(t *testing.T) {
	home := t.TempDir()
	setHomeForTests(t, home)
	setPasswrodForTests(t, "12345")

	s := newFakeServer(t)
	ts := httptest.NewServer(s.mux())
	t.Cleanup(ts.Close)

	app := cliapp.New(cliapp.BuildInfo{Version: "test", Date: "test", Commit: "test"})

	out := captureStdout(t, func() {
		err := app.Run([]string{
			"register",
			"--server", ts.URL,
			"--login", "test_user",
		})
		if err != nil {
			t.Fatalf("register failed: %v", err)
		}
	})
	if !strings.Contains(out, "registered user_id:") {
		t.Fatalf("expected register output to contain user id, got: %s", out)
	}

	out = captureStdout(t, func() {
		err := app.Run([]string{
			"login",
			"--server", ts.URL,
			"--login", "test_user",
		})
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}
	})
	if !strings.Contains(out, "login ok:") {
		t.Fatalf("expected login ok output, got: %s", out)
	}

	cfgPath := filepath.Join(home, ".gophkeeper", "config.json")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatalf("expected config.json saved, stat err: %v", err)
	}

	t.Setenv("GK_MASTER_PASS", "my-master-pass")
	out = captureStdout(t, func() {
		err := app.Run([]string{
			"add-text",
			"--title", "wifi",
			"--text", "pass=qwerty",
			"--meta", "home router",
		})
		if err != nil {
			t.Fatalf("add-text failed: %v", err)
		}
	})
	if !strings.Contains(out, "saved item id:") {
		t.Fatalf("expected saved item id output, got: %s", out)
	}
	itemID := extractUUID(t, out)

	out = captureStdout(t, func() {
		err := app.Run([]string{"sync", "--full"})
		if err != nil {
			t.Fatalf("sync failed: %v", err)
		}
	})
	if !strings.Contains(out, "sync ok; received:") {
		t.Fatalf("expected sync ok output, got: %s", out)
	}

	out = captureStdout(t, func() {
		err := app.Run([]string{"list"})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}
	})
	if !strings.Contains(out, itemID) || !strings.Contains(out, `title="wifi"`) {
		t.Fatalf("expected list output to include item id and title, got:\n%s", out)
	}

	out = captureStdout(t, func() {
		err := app.Run([]string{"get", "--id", itemID})
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
	})
	if !strings.Contains(out, `"text": "pass=qwerty"`) || !strings.Contains(out, `"metadata": "home router"`) {
		t.Fatalf("expected decrypted json payload in output, got:\n%s", out)
	}

	out = captureStdout(t, func() {
		err := app.Run([]string{"delete", "--id", itemID})
		if err != nil {
			t.Fatalf("delete failed: %v", err)
		}
	})
	if !strings.Contains(out, "deleted (tombstone saved)") || !strings.Contains(out, "version: 2") {
		t.Fatalf("expected delete output with version 2, got:\n%s", out)
	}

	out = captureStdout(t, func() {
		err := app.Run([]string{"sync"})
		if err != nil {
			t.Fatalf("sync2 failed: %v", err)
		}
	})
	out = captureStdout(t, func() {
		err := app.Run([]string{"list"})
		if err != nil {
			t.Fatalf("list2 failed: %v", err)
		}
	})
	if !strings.Contains(out, itemID) || !strings.Contains(out, "(deleted)") {
		t.Fatalf("expected deleted marker in list output, got:\n%s", out)
	}
}

func setHomeForTests(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func setPasswrodForTests(t *testing.T, pass string) {
	t.Helper()
	t.Setenv("GK_PASSWORD", "12345")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = orig
	return <-done
}

func extractUUID(t *testing.T, s string) string {
	t.Helper()
	re := regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
	m := re.FindString(s)
	if m == "" {
		t.Fatalf("uuid not found in: %q", s)
	}
	return m
}

type fakeServer struct {
	t  *testing.T
	mu sync.Mutex

	users     map[string]uuid.UUID
	passwords map[string]string
	kdfSalt   map[uuid.UUID][]byte

	tokens map[string]uuid.UUID

	items map[uuid.UUID]map[uuid.UUID]fakeItem
}

type fakeItem struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      string
	Encrypted []byte
	Metadata  string
	Version   int64
	Deleted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func newFakeServer(t *testing.T) *fakeServer {
	return &fakeServer{
		t:         t,
		users:     map[string]uuid.UUID{},
		passwords: map[string]string{},
		kdfSalt:   map[uuid.UUID][]byte{},
		tokens:    map[string]uuid.UUID{},
		items:     map[uuid.UUID]map[uuid.UUID]fakeItem{},
	}
}

func (s *fakeServer) mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/register", s.handleRegister)
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/items/", s.withAuth(s.handleItems))
	mux.HandleFunc("/items/sync", s.withAuth(s.handleSync))
	return mux
}

func (s *fakeServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if req.Login == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "login/password required"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[req.Login]; ok {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "user exists"})
		return
	}
	uid := uuid.New()
	s.users[req.Login] = uid
	s.passwords[req.Login] = req.Password

	rawSalt := make([]byte, 16)
	copy(rawSalt, []byte("0123456789abcdef"))
	s.kdfSalt[uid] = rawSalt

	writeJSON(w, http.StatusOK, map[string]any{"user_id": uid.String()})
}

func (s *fakeServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	uid, ok := s.users[req.Login]
	if !ok || s.passwords[req.Login] != req.Password {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid credentials"})
		return
	}

	token := "t_" + uuid.New().String()
	s.tokens[token] = uid

	saltB64 := base64.StdEncoding.EncodeToString(s.kdfSalt[uid])
	writeJSON(w, http.StatusOK, map[string]any{
		"token":        token,
		"kdf_salt_b64": saltB64,
	})
}

func (s *fakeServer) withAuth(next func(w http.ResponseWriter, r *http.Request, uid uuid.UUID)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
		if token == "" {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		s.mu.Lock()
		uid, ok := s.tokens[token]
		s.mu.Unlock()

		if !ok {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next(w, r, uid)
	}
}

func (s *fakeServer) handleItems(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	switch r.Method {
	case http.MethodPost:
		s.handleUpsertItem(w, r, uid)
	case http.MethodGet:
		s.handleListItems(w, r, uid)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (s *fakeServer) handleUpsertItem(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	var req struct {
		ID           string `json:"id,omitempty"`
		Type         string `json:"type"`
		EncryptedB64 string `json:"encrypted_b64,omitempty"`
		Metadata     string `json:"metadata,omitempty"`
		Version      int64  `json:"version,omitempty"`
		Deleted      bool   `json:"deleted,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if req.Type == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "type required"})
		return
	}

	var id uuid.UUID
	if req.ID == "" {
		id = uuid.New()
	} else {
		parsed, err := uuid.Parse(req.ID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad id"})
			return
		}
		id = parsed
	}

	if req.Version <= 0 {
		req.Version = 1
	}

	var enc []byte
	if req.Deleted {
		enc = []byte{}
	} else {
		if req.EncryptedB64 == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "encrypted_b64 required"})
			return
		}
		b, err := base64.StdEncoding.DecodeString(req.EncryptedB64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad encrypted_b64"})
			return
		}
		enc = b
	}

	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[uid]; !ok {
		s.items[uid] = map[uuid.UUID]fakeItem{}
	}
	prev, has := s.items[uid][id]
	if has {
		if prev.Version >= req.Version {
			writeJSON(w, http.StatusOK, toItemResp(prev))
			return
		}
		prev.Type = req.Type
		prev.Encrypted = enc
		prev.Metadata = req.Metadata
		prev.Version = req.Version
		prev.Deleted = req.Deleted
		prev.UpdatedAt = now
		s.items[uid][id] = prev
		writeJSON(w, http.StatusOK, toItemResp(prev))
		return
	}

	it := fakeItem{
		ID:        id,
		UserID:    uid,
		Type:      req.Type,
		Encrypted: enc,
		Metadata:  req.Metadata,
		Version:   req.Version,
		Deleted:   req.Deleted,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.items[uid][id] = it
	writeJSON(w, http.StatusOK, toItemResp(it))
}

func (s *fakeServer) handleListItems(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.items[uid]
	resp := make([]map[string]any, 0, len(m))
	for _, it := range m {
		resp = append(resp, toItemResp(it))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *fakeServer) handleSync(w http.ResponseWriter, r *http.Request, uid uuid.UUID) {
	sinceStr := r.URL.Query().Get("since")

	var since time.Time
	var err error
	if sinceStr == "" {
		since = time.Time{}
	} else {
		since, err = time.Parse(time.RFC3339Nano, sinceStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad since"})
			return
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	m := s.items[uid]
	resp := make([]map[string]any, 0, len(m))
	for _, it := range m {
		if it.UpdatedAt.After(since) {
			resp = append(resp, toItemResp(it))
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func toItemResp(it fakeItem) map[string]any {
	out := map[string]any{
		"id":         it.ID.String(),
		"type":       it.Type,
		"metadata":   it.Metadata,
		"version":    it.Version,
		"deleted":    it.Deleted,
		"created_at": it.CreatedAt,
		"updated_at": it.UpdatedAt,
	}
	if !it.Deleted {
		out["encrypted_b64"] = base64.StdEncoding.EncodeToString(it.Encrypted)
	} else {
		out["encrypted_b64"] = ""
	}
	return out
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
