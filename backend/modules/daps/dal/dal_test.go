package dal_test

import (
	"context"
	libdal "ftl/daps/dal"
	"testing"

	"github.com/TBD54566975/dap-go/dap"
	"github.com/TBD54566975/ftl/go-runtime/ftl"
	"github.com/TBD54566975/ftl/go-runtime/ftl/ftltest"
	"github.com/alecthomas/assert/v2"
	"go.jetpack.io/typeid"
)

func setup(t *testing.T) (context.Context, *libdal.DAL) {
	t.Helper()
	dbHandle := ftl.PostgresDatabase("daps")
	ctx := ftltest.Context(
		ftltest.WithDefaultProjectFile(),
		ftltest.WithDatabase(dbHandle),
	)

	db := dbHandle.Get(ctx)
	dal := libdal.New(db)

	t.Cleanup(func() {
		_ = db.Close()
	})

	return ctx, dal
}

func TestCreateDAP(t *testing.T) {
	t.Run("works", func(t *testing.T) {
		ctx, dal := setup(t)

		reg := dap.RegistrationRequest{
			ID:     typeid.Must(typeid.New[dap.RegistrationID]()),
			DID:    "did:example:123",
			Handle: "test",
		}

		err := dal.CreateDAP(ctx, reg)
		assert.NoError(t, err)
	})

	t.Run("dupe_did", func(t *testing.T) {
		ctx, dal := setup(t)

		reg := dap.RegistrationRequest{
			ID:     typeid.Must(typeid.New[dap.RegistrationID]()),
			DID:    "did:example:123",
			Handle: "beep",
		}

		err := dal.CreateDAP(ctx, reg)
		assert.NoError(t, err)

		reg2 := dap.RegistrationRequest{
			ID:     typeid.Must(typeid.New[dap.RegistrationID]()),
			DID:    "did:example:123",
			Handle: "boop",
		}

		err = dal.CreateDAP(ctx, reg2)
		assert.Error(t, err)
		assert.Equal(t, libdal.ErrDIDConflict, err)
	})

	t.Run("dupe_handle", func(t *testing.T) {
		ctx, dal := setup(t)

		reg := dap.RegistrationRequest{
			ID:     typeid.Must(typeid.New[dap.RegistrationID]()),
			DID:    "did:example:123",
			Handle: "beep",
		}

		err := dal.CreateDAP(ctx, reg)
		assert.NoError(t, err)

		reg2 := dap.RegistrationRequest{
			ID:     typeid.Must(typeid.New[dap.RegistrationID]()),
			DID:    "did:example:234",
			Handle: "beep",
		}

		err = dal.CreateDAP(ctx, reg2)
		assert.Error(t, err)
		assert.Equal(t, libdal.ErrHandleConflict, err)
	})
}
