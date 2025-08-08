package util

import "context"

func ExtractURLPath(ctx context.Context) string {
	val, _ := ctx.Value("urlPath").(string)

	return val
}
