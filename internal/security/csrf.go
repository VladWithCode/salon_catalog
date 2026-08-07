// Package security provides HTTP request protections for public application routes.
package security

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var (
	ErrCSRFTrustedOriginsMissing = errors.New("required env var CSRF_TRUSTED_ORIGINS is not set")
	ErrCSRFTrustedOriginsEmpty   = errors.New("CSRF_TRUSTED_ORIGINS must contain at least one origin")
	ErrCSRFTrustedOriginsInvalid = errors.New("CSRF_TRUSTED_ORIGINS contains an invalid origin")
)

// CSRFConfig contains the explicit public origins allowed to submit mutations.
type CSRFConfig struct {
	TrustedOrigins string
}

// CSRFGuard validates Origin or Referer before an unsafe request reaches a handler.
type CSRFGuard struct {
	trustedOrigins map[normalizedOrigin]struct{}
}

type normalizedOrigin struct {
	scheme   string
	hostname string
	port     string
}

// NewCSRFGuard validates configuration and constructs an immutable origin guard.
func NewCSRFGuard(config CSRFConfig) (*CSRFGuard, error) {
	rawOrigins := config.TrustedOrigins
	if strings.TrimSpace(rawOrigins) == "" {
		return nil, ErrCSRFTrustedOriginsEmpty
	}

	trustedOrigins := make(map[normalizedOrigin]struct{})
	for _, rawOrigin := range strings.Split(rawOrigins, ",") {
		rawOrigin = strings.TrimSpace(rawOrigin)
		if rawOrigin == "" {
			return nil, ErrCSRFTrustedOriginsInvalid
		}

		origin, err := parseNormalizedOrigin(rawOrigin, false)
		if err != nil {
			return nil, ErrCSRFTrustedOriginsInvalid
		}
		trustedOrigins[origin] = struct{}{}
	}

	if len(trustedOrigins) == 0 {
		return nil, ErrCSRFTrustedOriginsEmpty
	}
	return &CSRFGuard{trustedOrigins: trustedOrigins}, nil
}

// NewCSRFGuardFromEnv loads the trusted origins once during process startup.
func NewCSRFGuardFromEnv() (*CSRFGuard, error) {
	config, err := csrfConfigFromLookup(os.LookupEnv)
	if err != nil {
		return nil, err
	}
	return NewCSRFGuard(config)
}

func csrfConfigFromLookup(lookupEnv func(string) (string, bool)) (CSRFConfig, error) {
	rawOrigins, ok := lookupEnv("CSRF_TRUSTED_ORIGINS")
	if !ok {
		return CSRFConfig{}, ErrCSRFTrustedOriginsMissing
	}
	if strings.TrimSpace(rawOrigins) == "" {
		return CSRFConfig{}, ErrCSRFTrustedOriginsEmpty
	}
	return CSRFConfig{TrustedOrigins: rawOrigins}, nil
}

// Protect checks browser origin headers on POST, PUT, PATCH, and DELETE requests.
func (g *CSRFGuard) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isProtectedMethod(r.Method) {
			next(w, r)
			return
		}
		if !g.requestOriginIsTrusted(r) {
			writeCSRFRejection(w)
			return
		}
		next(w, r)
	}
}

func (g *CSRFGuard) requestOriginIsTrusted(r *http.Request) bool {
	originValues := r.Header.Values("Origin")
	if len(originValues) > 0 {
		if len(originValues) != 1 || strings.Contains(originValues[0], ",") {
			return false
		}
		origin, err := parseNormalizedOrigin(originValues[0], false)
		if err != nil {
			return false
		}
		_, trusted := g.trustedOrigins[origin]
		return trusted
	}

	refererValues := r.Header.Values("Referer")
	if len(refererValues) != 1 {
		return false
	}
	refererOrigin, err := parseNormalizedOrigin(refererValues[0], true)
	if err != nil {
		return false
	}
	_, trusted := g.trustedOrigins[refererOrigin]
	return trusted
}

func parseNormalizedOrigin(rawURL string, allowURLDetails bool) (normalizedOrigin, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || strings.EqualFold(rawURL, "null") || strings.Contains(rawURL, "*") {
		return normalizedOrigin{}, ErrCSRFTrustedOriginsInvalid
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Opaque != "" || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return normalizedOrigin{}, ErrCSRFTrustedOriginsInvalid
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return normalizedOrigin{}, ErrCSRFTrustedOriginsInvalid
	}
	if !allowURLDetails {
		if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || strings.Contains(rawURL, "#") {
			return normalizedOrigin{}, ErrCSRFTrustedOriginsInvalid
		}
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" || strings.HasSuffix(parsed.Host, ":") {
		return normalizedOrigin{}, ErrCSRFTrustedOriginsInvalid
	}

	port := parsed.Port()
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	} else {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return normalizedOrigin{}, ErrCSRFTrustedOriginsInvalid
		}
		port = strconv.Itoa(portNumber)
	}

	return normalizedOrigin{scheme: scheme, hostname: hostname, port: port}, nil
}

func isProtectedMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func writeCSRFRejection(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte("Solicitud no permitida."))
}
