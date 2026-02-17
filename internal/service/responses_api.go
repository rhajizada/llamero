package service

import "context"

// RouteResponsesCreate selects a backend for Responses API creation requests.
func (s *Service) RouteResponsesCreate(ctx context.Context, model string) (BackendRoute, error) {
	return s.RouteBackend(ctx, model)
}
