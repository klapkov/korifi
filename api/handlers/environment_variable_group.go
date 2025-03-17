package handlers

import (
	"net/http"
	"net/url"

	"code.cloudfoundry.org/korifi/api/presenter"
	"code.cloudfoundry.org/korifi/api/routing"
)

const (
	EnvVarGroupsPath = "/v3/environment_variable_groups"
	EnvVarGroupPath  = "/v3/environment_variable_groups/{name}"
)

type EnvVarGroup struct {
	serverURL url.URL
}

func NewEnvVarGroup(serverURL url.URL) *EnvVarGroup {
	return &EnvVarGroup{serverURL: serverURL}
}

func (h *EnvVarGroup) get(r *http.Request) (*routing.Response, error) {
	group := routing.URLParam(r, "name")
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForEnvVarGroup(h.serverURL, group)), nil
}

func (h *EnvVarGroup) list(r *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForEnvVarGroups(h.serverURL)), nil
}

func (h *EnvVarGroup) update(r *http.Request) (*routing.Response, error) {
	group := routing.URLParam(r, "name")
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForEnvVarGroup(h.serverURL, group)), nil
}

func (h *EnvVarGroup) UnauthenticatedRoutes() []routing.Route {
	return nil
}

func (h *EnvVarGroup) AuthenticatedRoutes() []routing.Route {
	return []routing.Route{
		{Method: "GET", Pattern: EnvVarGroupPath, Handler: h.get},
		{Method: "GET", Pattern: EnvVarGroupsPath, Handler: h.list},
		{Method: "PATCH", Pattern: EnvVarGroupPath, Handler: h.update},
	}
}
