package util

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/url"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal"
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

type WithMixedFragmentsConf struct {
	// FromTemplate is a templ.Component that will be rendered whole.
	// That is, no fragments will be selected, but instead the whole template will be included in the response.
	FromTemplate *templ.Component
	// JoinTemplates is a list of templ.Component that will be joined together
	// and from which the fragments will be selected.
	JoinTemplates []*templ.Component
	// Fragments is a list of fragments that will be selected from the JoinTemplates.
	Fragments []any
}

func (c *WithMixedFragmentsConf) SetFromTemplate(t templ.Component) *WithMixedFragmentsConf {
	c.FromTemplate = &t
	return c
}

func (c *WithMixedFragmentsConf) AppendJoinTemplates(t ...templ.Component) *WithMixedFragmentsConf {
	for _, t := range t {
		c.JoinTemplates = append(c.JoinTemplates, &t)
	}
	return c
}

func (c *WithMixedFragmentsConf) SetFragments(f ...any) *WithMixedFragmentsConf {
	c.Fragments = f
	return c
}

func RenderMixedWithFragments(ctx context.Context, w http.ResponseWriter, conf WithMixedFragmentsConf) {
	buf := bytes.NewBuffer(nil)
	var err error
	if len(conf.Fragments) > 0 {
		combined := templ.Join(internal.PtrSliceToPlainSlice(conf.JoinTemplates)...)
		templ.RenderFragments(ctx, buf, combined, conf.Fragments...)
	}
	if conf.FromTemplate != nil {
		(*conf.FromTemplate).Render(ctx, buf)
	}

	_, err = buf.WriteTo(w)
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
	}
}
