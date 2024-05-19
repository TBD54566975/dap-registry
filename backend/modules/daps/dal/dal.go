package dal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"ftl/daps/dal/sqlc"

	"github.com/TBD54566975/dap-go/dap"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrDIDConflict = errors.New("DID already exists")
var ErrHandleConflict = errors.New("handle already exists")
var ErrNotFound = errors.New("DAP not found")

type DAL struct {
	db    *sql.DB
	query sqlc.Querier
}

func New(conn *sql.DB) *DAL {
	return &DAL{db: conn, query: sqlc.New(conn)}
}

func (d *DAL) CreateDAP(ctx context.Context, reg dap.RegistrationRequest) error {
	proof, err := json.Marshal(reg)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}

	params := sqlc.CreateDAPParams{
		ID:     reg.ID.String(),
		Did:    reg.DID,
		Handle: reg.Handle,
		Proof:  proof,
	}

	if err := d.query.CreateDAP(ctx, params); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				if pgErr.ConstraintName == "daps_did_key" {
					return ErrDIDConflict
				}
				if pgErr.ConstraintName == "daps_handle_key" {
					return ErrHandleConflict
				}
			}
		}

		return err
	}

	return nil
}

func (d *DAL) GetHandleRegistration(ctx context.Context, handle string) (*dap.RegistrationRequest, error) {
	entry, err := d.query.GetDAP(ctx, handle)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	var reg dap.RegistrationRequest
	if err := json.Unmarshal(entry.Proof, &reg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal registration request: %w", err)
	}

	return &reg, nil
}
