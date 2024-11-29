package handlers

import (
	"net/http"
	"net/url"

	"code.cloudfoundry.org/korifi/api/presenter"
	"code.cloudfoundry.org/korifi/api/routing"
)

const (
	RootPath = "/"
)

type Root struct {
	baseURL url.URL
	uaaURL  url.URL
}

func NewRoot(baseURL url.URL, uaaURL url.URL) *Root {
	return &Root{
		baseURL: baseURL,
		uaaURL:  uaaURL,
	}
}

func (h *Root) get(r *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForRoot(h.baseURL, h.uaaURL)), nil
}

func (h *Root) UnauthenticatedRoutes() []routing.Route {
	return []routing.Route{
		{Method: "GET", Pattern: RootPath, Handler: h.get},
	}
}

func (h *Root) AuthenticatedRoutes() []routing.Route {
	return nil
}
