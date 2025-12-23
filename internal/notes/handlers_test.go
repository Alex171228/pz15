package notes

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubStore struct {
	createFn   func(context.Context, string, string) (Note, error)
	getFn      func(context.Context, int64) (Note, error)
	updateFn   func(context.Context, int64, string, string) (Note, error)
	deleteFn   func(context.Context, int64) error
	listFn     func(context.Context, ListParams) ([]Note, error)
	batchGetFn func(context.Context, []int64) ([]Note, error)
}

func (s stubStore) Create(ctx context.Context, title, content string) (Note, error) {
	return s.createFn(ctx, title, content)
}
func (s stubStore) Get(ctx context.Context, id int64) (Note, error) {
	return s.getFn(ctx, id)
}
func (s stubStore) Update(ctx context.Context, id int64, title, content string) (Note, error) {
	return s.updateFn(ctx, id, title, content)
}
func (s stubStore) Delete(ctx context.Context, id int64) error             { return s.deleteFn(ctx, id) }
func (s stubStore) List(ctx context.Context, p ListParams) ([]Note, error) { return s.listFn(ctx, p) }
func (s stubStore) BatchGet(ctx context.Context, ids []int64) ([]Note, error) {
	return s.batchGetFn(ctx, ids)
}

func TestHandlers_Create_Validation(t *testing.T) {
	h := NewHandlers(stubStore{
		createFn: func(context.Context, string, string) (Note, error) {
			return Note{}, nil
		},
	}).Routes()

	req := httptest.NewRequest(http.MethodPost, "/notes/", bytes.NewBufferString(`{"title":"","content":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlers_Create_Success(t *testing.T) {
	created := Note{ID: 1, Title: "t", Content: "c", CreatedAt: time.Unix(1, 0).UTC()}
	h := NewHandlers(stubStore{
		createFn: func(context.Context, string, string) (Note, error) {
			return created, nil
		},
	}).Routes()

	req := httptest.NewRequest(http.MethodPost, "/notes/", bytes.NewBufferString(`{"title":"t","content":"c"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	var got Note
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	require.Equal(t, created.ID, got.ID)
}

func TestHandlers_Get_InvalidID(t *testing.T) {
	h := NewHandlers(stubStore{
		getFn: func(context.Context, int64) (Note, error) { return Note{}, nil },
	}).Routes()

	req := httptest.NewRequest(http.MethodGet, "/notes/abc", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestHandlers_Get_Success_NotFound_And_Internal(t *testing.T) {
	n := Note{ID: 42, Title: "t", Content: "c", CreatedAt: time.Unix(2, 0).UTC()}

	// success
	{
		h := NewHandlers(stubStore{
			getFn: func(context.Context, int64) (Note, error) { return n, nil },
		}).Routes()
		req := httptest.NewRequest(http.MethodGet, "/notes/42", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	// not found
	{
		h := NewHandlers(stubStore{
			getFn: func(context.Context, int64) (Note, error) { return Note{}, sql.ErrNoRows },
		}).Routes()
		req := httptest.NewRequest(http.MethodGet, "/notes/999", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusNotFound, rr.Code)
	}

	// internal error
	{
		boom := errors.New("boom")
		h := NewHandlers(stubStore{
			getFn: func(context.Context, int64) (Note, error) { return Note{}, boom },
		}).Routes()
		req := httptest.NewRequest(http.MethodGet, "/notes/1", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusInternalServerError, rr.Code)
	}
}

func TestHandlers_Update_Delete_And_List(t *testing.T) {
	fixed := time.Unix(3, 0).UTC()

	store := stubStore{
		updateFn: func(context.Context, int64, string, string) (Note, error) {
			return Note{ID: 1, Title: "t2", Content: "c2", CreatedAt: fixed}, nil
		},
		deleteFn: func(context.Context, int64) error { return nil },
		listFn: func(_ context.Context, p ListParams) ([]Note, error) {
			require.Equal(t, 10, p.Limit)
			require.Equal(t, "title", p.Query)
			return []Note{{ID: 2, Title: "a", Content: "b", CreatedAt: fixed}}, nil
		},
		batchGetFn: func(context.Context, []int64) ([]Note, error) { return []Note{}, nil },
		createFn:   func(context.Context, string, string) (Note, error) { return Note{}, nil },
		getFn:      func(context.Context, int64) (Note, error) { return Note{}, nil },
	}

	h := NewHandlers(store).Routes()

	// update invalid json
	{
		req := httptest.NewRequest(http.MethodPut, "/notes/1", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	}

	// update success
	{
		req := httptest.NewRequest(http.MethodPut, "/notes/1", bytes.NewBufferString(`{"title":"t2","content":"c2"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	// delete success
	{
		req := httptest.NewRequest(http.MethodDelete, "/notes/1", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusNoContent, rr.Code)
	}

	// list parses params + next cursor
	{
		req := httptest.NewRequest(http.MethodGet, "/notes?limit=10&q=title", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
		var resp map[string]any
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
		require.Equal(t, float64(2), resp["next_cursor_id"]) // JSON numbers decode to float64
	}
}

func TestHandlers_Batch_InvalidJSON_And_Success(t *testing.T) {
	store := stubStore{
		batchGetFn: func(context.Context, []int64) ([]Note, error) {
			return []Note{{ID: 1, Title: "t", Content: "c", CreatedAt: time.Unix(4, 0).UTC()}}, nil
		},
		createFn: func(context.Context, string, string) (Note, error) { return Note{}, nil },
		getFn:    func(context.Context, int64) (Note, error) { return Note{}, nil },
		updateFn: func(context.Context, int64, string, string) (Note, error) { return Note{}, nil },
		deleteFn: func(context.Context, int64) error { return nil },
		listFn:   func(context.Context, ListParams) ([]Note, error) { return []Note{}, nil },
	}
	h := NewHandlers(store).Routes()

	// invalid json
	{
		req := httptest.NewRequest(http.MethodPost, "/notes/batch", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusBadRequest, rr.Code)
	}

	// success
	{
		req := httptest.NewRequest(http.MethodPost, "/notes/batch", bytes.NewBufferString(`{"ids":[1]}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}
}
