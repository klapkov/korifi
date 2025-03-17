package handlers

import (
	"net/http"
	"net/url"

	"code.cloudfoundry.org/korifi/api/presenter"
	"code.cloudfoundry.org/korifi/api/routing"
)

const (
	FeatureFlagsPath = "/v3/feature_flags"
	FeatureFlagPath  = "/v3/feature_flags/{name}"
)

type FeatureFlag struct {
	serverURL url.URL
}

func NewFeatureFlag(serverURL url.URL) *FeatureFlag {
	return &FeatureFlag{serverURL: serverURL}
}

func (h *FeatureFlag) get(r *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForFeatureFlag(h.serverURL)), nil
}

func (h *FeatureFlag) list(r *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForFeatureFlags(h.serverURL)), nil
}

func (h *FeatureFlag) update(r *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForFeatureFlag(h.serverURL)), nil
}

func (h *FeatureFlag) UnauthenticatedRoutes() []routing.Route {
	return nil
}

func (h *FeatureFlag) AuthenticatedRoutes() []routing.Route {
	return []routing.Route{
		{Method: "GET", Pattern: FeatureFlagPath, Handler: h.get},
		{Method: "GET", Pattern: FeatureFlagsPath, Handler: h.list},
		{Method: "PATCH", Pattern: FeatureFlagPath, Handler: h.update},
	}
}
