package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	itemsvc "github.com/JinFuuMugen/GophKeeper/internal/items/service"
	"github.com/JinFuuMugen/GophKeeper/internal/middleware"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

type ItemsHandler struct {
	svc    *itemsvc.Service
	logger *slog.Logger
}

func NewItemsHandler(svc *itemsvc.Service, logger *slog.Logger) *ItemsHandler {
	return &ItemsHandler{svc: svc, logger: logger}
}

type upsertItemReq struct {
	ID           string `json:"id,omitempty"`
	Type         string `json:"type"`
	EncryptedB64 string `json:"encrypted_b64,omitempty"`
	Metadata     string `json:"metadata,omitempty"`
	Version      int64  `json:"version,omitempty"`
	Deleted      bool   `json:"deleted,omitempty"`
}

type itemResp struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	EncryptedB64 string    `json:"encrypted_b64,omitempty"`
	Metadata     string    `json:"metadata"`
	Version      int64     `json:"version"`
	Deleted      bool      `json:"deleted"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toItemResp(it models.Item) itemResp {
	resp := itemResp{
		ID:        it.ID.String(),
		Type:      it.Type,
		Metadata:  it.Metadata,
		Version:   it.Version,
		Deleted:   it.Deleted,
		CreatedAt: it.CreatedAt,
		UpdatedAt: it.UpdatedAt,
	}
	if !it.Deleted && len(it.Encrypted) > 0 {
		resp.EncryptedB64 = base64.StdEncoding.EncodeToString(it.Encrypted)
	}

	return resp
}

func (h *ItemsHandler) UpsertItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req upsertItemReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	var id uuid.UUID
	if req.ID != "" {
		parsed, err := uuid.Parse(req.ID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid id")
			return
		}
		id = parsed
	}

	var encrypted []byte
	if !req.Deleted {
		if req.EncryptedB64 == "" {
			WriteError(w, http.StatusBadRequest, "encrypted_b64 required")
			return
		}
		b, err := base64.StdEncoding.DecodeString(req.EncryptedB64)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "invalid encrypted_b64 (expected base64)")
			return
		}
		encrypted = b
	}

	saved, err := h.svc.Upsert(r.Context(), models.Item{
		ID:        id,
		UserID:    userID,
		Type:      req.Type,
		Encrypted: encrypted,
		Metadata:  req.Metadata,
		Version:   req.Version,
		Deleted:   req.Deleted,
	})
	if err != nil {
		h.logger.Error("upsert item failed", "error", err)
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, toItemResp(saved))
}

func (h *ItemsHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	items, err := h.svc.List(r.Context(), userID)
	if err != nil {
		h.logger.Error("list items failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]itemResp, 0, len(items))
	for _, it := range items {
		resp = append(resp, toItemResp(it))
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *ItemsHandler) SyncItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sinceStr := r.URL.Query().Get("since")
	if sinceStr == "" {
		h.ListItems(w, r)
		return
	}

	since, err := time.Parse(time.RFC3339Nano, sinceStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad since value, expected RFC3339Nano")
		return
	}

	items, err := h.svc.SyncSince(r.Context(), userID, since)
	if err != nil {
		h.logger.Error("sync items failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]itemResp, 0, len(items))
	for _, it := range items {
		resp = append(resp, toItemResp(it))
	}
	WriteJSON(w, http.StatusOK, resp)
}
