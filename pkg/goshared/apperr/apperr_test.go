package apperr_test

import (
	"errors"
	"testing"

	"github.com/lawtrann/monorepo/pkg/goshared/apperr"
	"github.com/stretchr/testify/assert"
)

func TestNotFound(t *testing.T) {
	cause := errors.New("db error")
	e := &apperr.NotFound{Description: "user not found", Err: cause}
	assert.Equal(t, "user not found", e.Error())
	assert.Equal(t, cause, e.Unwrap())

	var target *apperr.NotFound
	assert.True(t, errors.As(e, &target))
}

func TestAlreadyExists(t *testing.T) {
	cause := errors.New("unique violation")
	e := &apperr.AlreadyExists{Description: "email taken", Err: cause}
	assert.Equal(t, "email taken", e.Error())
	assert.Equal(t, cause, e.Unwrap())

	var target *apperr.AlreadyExists
	assert.True(t, errors.As(e, &target))
}

func TestInvalidInput(t *testing.T) {
	e := &apperr.InvalidInput{Description: "bad request"}
	assert.Equal(t, "bad request", e.Error())
	assert.Nil(t, e.Unwrap())
}

func TestForbidden(t *testing.T) {
	e := &apperr.Forbidden{Description: "access denied"}
	assert.Equal(t, "access denied", e.Error())
	assert.Nil(t, e.Unwrap())
}

func TestConflict(t *testing.T) {
	cause := errors.New("concurrent edit")
	e := &apperr.Conflict{Description: "version mismatch", Err: cause}
	assert.Equal(t, "version mismatch", e.Error())
	assert.Equal(t, cause, e.Unwrap())

	var target *apperr.Conflict
	assert.True(t, errors.As(e, &target))
}

func TestErrorsIsWrapping(t *testing.T) {
	sentinel := errors.New("sentinel")
	e := &apperr.NotFound{Description: "not found", Err: sentinel}
	assert.True(t, errors.Is(e, sentinel))
}
