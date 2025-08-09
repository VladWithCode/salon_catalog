package util

import (
	"context"
	"net/url"
)

func ExtractURLPath(ctx context.Context) string {
	val, _ := ctx.Value("urlPath").(string)

	return val
}

func ExtractURLQuery(ctx context.Context) url.Values {
	val, ok := ctx.Value("urlQueryParams").(url.Values)
	if !ok {
		val = url.Values{}
	}

	return val
}

func ExtractEncodedURLQuery(ctx context.Context) string {
	q, ok := ctx.Value("urlQueryParams").(url.Values)
	if !ok {
		return ""
	}
	return q.Encode()
}
