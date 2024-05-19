package didweb

import (
	"context"
	"encoding/json"
	"fmt"
	"ftl/builtin"
	"net/http"
	"strings"

	"github.com/TBD54566975/ftl/go-runtime/ftl"
	"github.com/tbd54566975/web5-go/dids/did"
	"github.com/tbd54566975/web5-go/dids/didcore"
)

var secHandle = ftl.Secret[string]("did_web_portable_did")
var didDocument = ftl.Map(secHandle, func(ctx context.Context, sec string) (didcore.Document, error) {
	var pdid did.PortableDID
	if err := json.Unmarshal([]byte(sec), &pdid); err != nil {
		return didcore.Document{}, fmt.Errorf("failed to unmarshal portable did: %w", err)
	}

	return pdid.Document, nil
})

type ResolveRequest = builtin.HttpRequest[ftl.Unit]
type ResolveResponse = builtin.HttpResponse[[]byte, ftl.Unit]

//ftl:ingress http GET /did.json
func Resolve(ctx context.Context, req ResolveRequest) (ResolveResponse, error) {
	dd := didDocument.Get(ctx)
	if strings.Count(dd.ID, ":") <= 2 {
		return ResolveResponse{
			Status: http.StatusNotFound,
			Error:  ftl.Some(ftl.Unit{}),
		}, nil
	}

	m, _ := json.Marshal(dd)
	return ResolveResponse{
		Status: http.StatusOK,
		Body:   ftl.Some(m),
	}, nil
}

//ftl:ingress http GET /.well-known/did.json
func ResolveWellKnown(ctx context.Context, req ResolveRequest) (ResolveResponse, error) {
	dd := didDocument.Get(ctx)
	if strings.Count(dd.ID, ":") != 2 {
		return ResolveResponse{
			Status: http.StatusNotFound,
			Error:  ftl.Some(ftl.Unit{}),
		}, nil
	}

	m, _ := json.Marshal(dd)
	return ResolveResponse{
		Status: http.StatusOK,
		Body:   ftl.Some(m),
	}, nil
}
