package daps

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"ftl/builtin"
	"net/http"

	_dal "ftl/daps/dal"

	libdap "github.com/TBD54566975/dap-go/dap"
	"github.com/TBD54566975/ftl/go-runtime/ftl"
	"github.com/tbd54566975/web5-go/dids/did"
)

var dbHandle = ftl.PostgresDatabase("daps")
var dal = ftl.Map(dbHandle, func(ctx context.Context, db *sql.DB) (*_dal.DAL, error) {
	return _dal.New(db), nil
})

var portableDIDHandle = ftl.Secret[string]("did_web_portable_did")
var bearerDIDHandle = ftl.Map(portableDIDHandle, func(ctx context.Context, sec string) (did.BearerDID, error) {
	var pd did.PortableDID
	err := json.Unmarshal([]byte(sec), &pd)
	if err != nil {
		return did.BearerDID{}, fmt.Errorf("failed to unmarshal portable DID: %w", err)
	}

	bearerDID, err := did.FromPortableDID(pd)
	if err != nil {
		return did.BearerDID{}, fmt.Errorf("failed to import bearer DID: %w", err)
	}

	return bearerDID, nil
})

//ftl:ingress http POST /daps
func Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	logger := ftl.LoggerFromContext(ctx)

	marshaled, err := json.Marshal(req.Body)
	if err != nil {
		return RegisterResponse{
			Status: http.StatusBadRequest,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusBadRequest,
				Message: "failed to marshal request body",
			}),
		}, nil
	}

	var reg libdap.RegistrationRequest
	if err := json.Unmarshal(marshaled, &reg); err != nil {
		return RegisterResponse{
			Status: http.StatusBadRequest,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusBadRequest,
				Message: "failed to unmarshal request body",
			}),
		}, nil
	}

	decodedJWS, err := reg.Verify()
	if err != nil {

		return RegisterResponse{
			Status: http.StatusUnauthorized,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusUnauthorized,
				Message: fmt.Sprintf("invalid signature: %s", err.Error()),
			}),
		}, nil
	}

	bdid := bearerDIDHandle.Get(ctx)
	signer := decodedJWS.SignerDID.URI
	if signer != reg.DID && signer != bdid.DID.String() {
		return RegisterResponse{
			Status: http.StatusBadRequest,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusBadRequest,
				Message: "invalid signature. signer DID does not match the provided DID or registry's DID",
			}),
		}, nil
	}

	if err := dal.Get(ctx).CreateDAP(ctx, reg); err != nil {
		if errors.Is(err, _dal.ErrDIDConflict) {
			return RegisterResponse{
				Status: http.StatusConflict,
				Error: ftl.Some(ErrResponse{
					Status:  http.StatusConflict,
					Message: "DID already registered",
				}),
			}, nil
		}

		if errors.Is(err, _dal.ErrHandleConflict) {
			return RegisterResponse{
				Status: http.StatusConflict,
				Error: ftl.Some(ErrResponse{
					Status:  http.StatusConflict,
					Message: "Handle already registered",
				}),
			}, nil
		}

		logger.Errorf(err, "failed to write DAP to db")

		return RegisterResponse{
			Status: http.StatusInternalServerError,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusInternalServerError,
				Message: "failed to process request",
			}),
		}, nil
	}

	//! TODO: sign the registration request digest and return it in the response

	return RegisterResponse{
		Status: http.StatusCreated,
		Body:   ftl.Some(RegisterResponseBody{}),
	}, nil
}

//ftl:ingress http GET /daps/{handle}
func Resolve(ctx context.Context, req ResolveRequest) (ResolveResponse, error) {
	logger := ftl.LoggerFromContext(ctx)

	handle, ok := req.PathParameters["handle"]
	if !ok {
		return ResolveResponse{
			Status: http.StatusBadRequest,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusBadRequest,
				Message: "expected handle in path",
			}),
		}, nil
	}

	reg, err := dal.Get(ctx).GetHandleRegistration(ctx, handle)
	if err != nil {
		logger.Errorf(err, "failed to query db for registration request. handle: %s", handle)

		return ResolveResponse{
			Status: http.StatusInternalServerError,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusInternalServerError,
				Message: "failed to process request",
			}),
		}, nil
	}

	if reg == nil {
		return ResolveResponse{
			Status: http.StatusNotFound,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusNotFound,
				Message: "handle not found",
			}),
		}, nil
	}

	marshaled, err := json.Marshal(reg)
	if err != nil {
		logger.Errorf(err, "failed to marshal registration request for handle: %v", handle)
		return ResolveResponse{
			Status: http.StatusInternalServerError,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusInternalServerError,
				Message: "failed to process request",
			}),
		}, nil
	}

	var resp RegisterRequestBody
	if err := json.Unmarshal(marshaled, &resp); err != nil {
		logger.Errorf(err, "failed to unmarshal registration request for handle: %v", handle)
		return ResolveResponse{
			Status: http.StatusInternalServerError,
			Error: ftl.Some(ErrResponse{
				Status:  http.StatusInternalServerError,
				Message: "failed to process request",
			}),
		}, nil
	}

	return ResolveResponse{
		Status: http.StatusOK,
		Body:   ftl.Some(resp),
	}, nil
}

type RegisterRequest = builtin.HttpRequest[RegisterRequestBody]
type RegisterRequestBody struct {
	ID        string `json:"id"`
	Handle    string `json:"handle"`
	DID       string `json:"did"`
	Domain    string `json:"domain"`
	Signature string `json:"signature"`
}

type RegisterResponse = builtin.HttpResponse[RegisterResponseBody, ErrResponse]
type RegisterResponseBody struct{}

type ResolveRequest = builtin.HttpRequest[ResolveRequestParams]
type ResolveRequestParams struct {
	Handle string
}

type ResolveResponse = builtin.HttpResponse[ResolveResponseBody, ErrResponse]
type ResolveResponseBody = RegisterRequestBody

type ErrResponse struct {
	Status  int    `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}
