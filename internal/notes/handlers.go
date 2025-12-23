package notes

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	store Store
}

// Store is an abstraction over the notes storage.
// It allows unit-testing handlers without a real database.
type Store interface {
	Create(ctx context.Context, title, content string) (Note, error)
	Get(ctx context.Context, id int64) (Note, error)
	Update(ctx context.Context, id int64, title, content string) (Note, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, p ListParams) ([]Note, error)
	BatchGet(ctx context.Context, ids []int64) ([]Note, error)
}

func NewHandlers(store Store) *Handlers {
	return &Handlers{store: store}
}

func (h *Handlers) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/notes", func(r chi.Router) {
		r.Post("/", h.create)
		r.Get("/", h.list)
		r.Post("/batch", h.batch)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.get)
			r.Put("/", h.update)
			r.Delete("/", h.delete)
		})
	})

	return r
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Title == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and content required"})
		return
	}

	n, err := h.store.Create(r.Context(), req.Title, req.Content)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	n, err := h.store.Get(r.Context(), id)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Title == "" || req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and content required"})
		return
	}

	n, err := h.store.Update(r.Context(), id, req.Title, req.Content)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handlers) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.store.Delete(r.Context(), id); err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")

	limit := 20
	if s := r.URL.Query().Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			limit = v
		}
	}

	var cursorAt *time.Time
	var cursorID *int64

	if s := r.URL.Query().Get("cursor_created_at"); s != "" {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			cursorAt = &t
		}
	}
	if s := r.URL.Query().Get("cursor_id"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			cursorID = &v
		}
	}

	items, err := h.store.List(r.Context(), ListParams{
		Limit:           limit,
		CursorCreatedAt: cursorAt,
		CursorID:        cursorID,
		Query:           q,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := map[string]any{"items": items}
	if len(items) > 0 {
		last := items[len(items)-1]
		resp["next_cursor_created_at"] = last.CreatedAt.Format(time.RFC3339Nano)
		resp["next_cursor_id"] = last.ID
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) batch(w http.ResponseWriter, r *http.Request) {
	var req BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	items, err := h.store.BatchGet(r.Context(), req.IDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
