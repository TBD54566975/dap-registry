package daps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	dapsdk "github.com/TBD54566975/dap-go/dap"
	"github.com/TBD54566975/ftl/go-runtime/ftl/ftltest"
	"github.com/alecthomas/assert/v2"
	"github.com/tbd54566975/web5-go/dids/didjwk"
)

func setup() context.Context {
	return ftltest.Context(
		ftltest.WithDefaultProjectFile(),
		ftltest.WithDatabase(dbHandle),
		ftltest.WithMapsAllowed(),
	)
}

func TestRegister(t *testing.T) {
	t.Run("works", func(t *testing.T) {
		ctx := setup()

		bdid, err := didjwk.Create()
		assert.NoError(t, err)

		reg := dapsdk.NewRegistration("moegrammer", "didpay.me", bdid.URI)
		err = reg.Sign(bdid)
		assert.NoError(t, err)

		resp, err := Register(ctx, RegisterRequest{
			Method: http.MethodPost,
			Path:   "/daps",
			Body: RegisterRequestBody{
				ID:        reg.ID.String(),
				Handle:    reg.Handle,
				DID:       reg.DID,
				Domain:    reg.Domain,
				Signature: reg.Signature,
			},
		})

		assert.NoError(t, err)

		assert.NoError(t, err)
		j, err := json.MarshalIndent(resp.Error, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(j))

		assert.Equal(t, http.StatusCreated, resp.Status)
		assert.Zero(t, resp.Error)
	})
}

func TestResolve(t *testing.T) {
	t.Run("works", func(t *testing.T) {
		ctx := setup()

		bdid, err := didjwk.Create()
		assert.NoError(t, err)

		handle := "moegrammer"
		domain := "didpay.me"

		reg := dapsdk.NewRegistration(handle, domain, bdid.URI)
		err = reg.Sign(bdid)
		assert.NoError(t, err)

		regResp, err := Register(ctx, RegisterRequest{
			Method: http.MethodPost,
			Path:   "/daps",
			Body: RegisterRequestBody{
				ID:        reg.ID.String(),
				Handle:    reg.Handle,
				DID:       reg.DID,
				Domain:    reg.Domain,
				Signature: reg.Signature,
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, regResp.Status)

		resp, err := Resolve(ctx, ResolveRequest{
			Method: http.MethodGet,
			Path:   "/daps",
			PathParameters: map[string]string{
				"handle": handle,
			},
			Headers: map[string][]string{
				"Host": {domain},
			},
			Body: ResolveRequestParams{Handle: handle},
		})

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.Status)

		assert.Zero(t, resp.Error)
		assert.NotZero(t, resp.Body)

		body := resp.Body.MustGet()
		marshaled, err := json.Marshal(body)
		assert.NoError(t, err)

		var resolvedReg dapsdk.RegistrationRequest
		err = json.Unmarshal(marshaled, &resolvedReg)
		assert.NoError(t, err)

		expectedDigest, err := reg.Digest()
		assert.NoError(t, err)

		resolvedDigest, err := resolvedReg.Digest()
		assert.NoError(t, err)

		assert.Equal(t, expectedDigest, resolvedDigest)
	})
	t.Run("not_found", func(t *testing.T) {
		ctx := setup()

		resp, err := Resolve(ctx, ResolveRequest{
			Method: http.MethodGet,
			Path:   "/daps",
			PathParameters: map[string]string{
				"handle": "moegrammer",
			},
			Headers: map[string][]string{
				"Host": {"didpay.me"},
			},
			Body: ResolveRequestParams{Handle: "moegrammer"},
		})

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.Status)

		assert.Zero(t, resp.Body)
		assert.NotZero(t, resp.Error)

		body := resp.Error.MustGet()
		assert.Equal(t, body.Status, http.StatusNotFound)
		assert.NotZero(t, body.Message)
	})
}
