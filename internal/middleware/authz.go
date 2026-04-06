package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/rhajizada/llamero/internal/auth"
	"github.com/rhajizada/llamero/internal/httpjson"
	"github.com/rhajizada/llamero/internal/service"
	"github.com/rhajizada/llamero/internal/xslices"
)

const bearerTokenParts = 2

// Authz applies JWT verification and scope enforcement to HTTP handlers.
type Authz struct {
	verifier     *auth.TokenVerifier
	patValidator PATValidator
}

// PATValidator checks whether a PAT is still active.
type PATValidator interface {
	ValidatePAT(ctx context.Context, claims *auth.Claims) error
}

// NewAuthz constructs a scope-aware middleware set.
func NewAuthz(verifier *auth.TokenVerifier, patValidator PATValidator) *Authz {
	return &Authz{
		verifier:     verifier,
		patValidator: patValidator,
	}
}

// Require ensures the incoming request bears a token containing every supplied scope.
func (a *Authz) Require(scopes ...string) func(http.Handler) http.Handler {
	required := xslices.UniqueTrimmedStrings(scopes)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := authenticate(w, r, a.verifier, a.patValidator)
			if !ok {
				return
			}

			if !claims.HasScopes(required) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient scope"})
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext extracts auth.Claims stored by the middleware.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey{}).(*auth.Claims)
	return claims, ok
}

func ContextWithClaims(ctx context.Context, claims *auth.Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}

type claimsKey struct{}

var errPATValidationUnavailable = errors.New("pat validation unavailable")

var ErrPATValidationUnavailable = errPATValidationUnavailable

func ValidatePAT(ctx context.Context, claims *auth.Claims, patValidator PATValidator) error {
	return validatePAT(ctx, claims, patValidator)
}

func BearerToken(header string) string {
	return bearerToken(header)
}

func authenticate(
	w http.ResponseWriter,
	r *http.Request,
	verifier *auth.TokenVerifier,
	patValidator PATValidator,
) (*auth.Claims, bool) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return nil, false
	}

	claims, err := verifier.Verify(r.Context(), token)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return nil, false
	}

	if err = validatePAT(r.Context(), claims, patValidator); err != nil {
		var appErr *service.Error
		switch {
		case errors.As(err, &appErr):
			writeJSON(w, appErr.Code, map[string]string{"error": appErr.Message})
		case errors.Is(err, errPATValidationUnavailable):
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		default:
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		}
		return nil, false
	}

	return claims, true
}

func validatePAT(ctx context.Context, claims *auth.Claims, patValidator PATValidator) error {
	if claims.Type != auth.TokenTypePAT {
		return nil
	}

	if patValidator == nil {
		return errPATValidationUnavailable
	}

	return patValidator.ValidatePAT(ctx, claims)
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", bearerTokenParts)
	if len(parts) != bearerTokenParts {
		return ""
	}
	if !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	httpjson.Write(w, status, payload)
}
