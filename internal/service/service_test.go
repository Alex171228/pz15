package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubRepo struct {
	byEmailFn func(email string) (User, error)
	createFn  func(email string) (User, error)
}

func (s stubRepo) ByEmail(email string) (User, error) {
	return s.byEmailFn(email)
}

func (s stubRepo) Create(email string) (User, error) {
	return s.createFn(email)
}

func TestService_FindIDByEmail(t *testing.T) {
	svc := New(stubRepo{
		byEmailFn: func(email string) (User, error) {
			require.Equal(t, "a@b", email)
			return User{ID: 42, Email: email}, nil
		},
		createFn: func(email string) (User, error) { return User{}, nil },
	})

	// invalid email branch
	_, err := svc.FindIDByEmail(" ")
	require.ErrorIs(t, err, ErrInvalidEmail)

	// ok branch
	id, err := svc.FindIDByEmail(" a@b ")
	require.NoError(t, err)
	require.Equal(t, int64(42), id)
}

func TestService_Register(t *testing.T) {
	t.Run("invalid email", func(t *testing.T) {
		svc := New(stubRepo{
			byEmailFn: func(email string) (User, error) { return User{}, ErrNotFound },
			createFn:  func(email string) (User, error) { return User{ID: 1, Email: email}, nil },
		})
		_, err := svc.Register("bad")
		require.ErrorIs(t, err, ErrInvalidEmail)
	})

	t.Run("already exists", func(t *testing.T) {
		svc := New(stubRepo{
			byEmailFn: func(email string) (User, error) { return User{ID: 7, Email: email}, nil },
			createFn:  func(email string) (User, error) { return User{ID: 999, Email: email}, nil },
		})
		_, err := svc.Register("x@y")
		require.ErrorIs(t, err, ErrAlreadyExists)
	})

	t.Run("repo error on lookup", func(t *testing.T) {
		boom := errors.New("boom")
		svc := New(stubRepo{
			byEmailFn: func(email string) (User, error) { return User{}, boom },
			createFn:  func(email string) (User, error) { return User{ID: 999, Email: email}, nil },
		})
		_, err := svc.Register("x@y")
		require.ErrorIs(t, err, boom)
	})

	t.Run("create success after not found", func(t *testing.T) {
		svc := New(stubRepo{
			byEmailFn: func(email string) (User, error) { return User{}, ErrNotFound },
			createFn: func(email string) (User, error) {
				require.Equal(t, "x@y", email)
				return User{ID: 100, Email: email}, nil
			},
		})
		u, err := svc.Register(" x@y ")
		require.NoError(t, err)
		require.Equal(t, int64(100), u.ID)
	})
}
