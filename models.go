package typesafe

import (
	"context"
	"net/http"
)

type Model struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReleaseDate string `json:"release_date"`
}

const (
	ModelsPath = "/v1/models"
)

func (c *Client) Models(ctx context.Context) ([]Model, error) {
	var resp struct {
		Models []Model `json:"models"`
	}
	if err := c.do(ctx, http.MethodGet, ModelsPath, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Models, nil
}
