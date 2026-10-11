package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kausys/apikit"

	"example.com/wrappers/dto"
)

// CreateRequest is the body under any json options.
type CreateRequest struct {
	Body CreateBody `json:"body,omitempty"`
}

// WebhookRequest takes the body as it came.
type WebhookRequest struct {
	RawBody []byte
}

// apikit:handler
func List(ctx context.Context, req ListRequest) (map[string]any, error) {
	return map[string]any{
		"limit":  *req.Limit,
		"active": req.Active != nil && *req.Active,
		"name":   req.Name,
		"status": req.Status,
		"kinds":  req.Kinds,
		"sizes":  req.Sizes,
		"since":  req.Since,
		"trace":  req.Trace,
	}, nil
}

// apikit:handler
func Get(ctx context.Context, req GetRequest) (string, error) {
	if req.ID.String() == "itm_missing" {
		return "", apikit.NotFound("item")
	}
	return req.ID.String(), nil
}

// apikit:handler
func Create(ctx context.Context, req CreateRequest) (*apikit.HttpResponse, error) {
	return apikit.NewHttpResponse(http.StatusCreated, req.Body), nil
}

// apikit:handler
func Webhook(ctx context.Context, req WebhookRequest) (string, error) {
	return fmt.Sprint(len(req.RawBody)), nil
}

// apikit:handler
func Rename(ctx context.Context, req dto.RenameRequest) (string, error) {
	return req.ID + ":" + req.Body.Name, nil
}
