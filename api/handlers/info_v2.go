package handlers

import (
	"net/http"
	"net/url"

	"code.cloudfoundry.org/korifi/api/presenter"
	"code.cloudfoundry.org/korifi/api/routing"
)

const V2InfoPath = "/v2/info"

type InfoV2 struct {
	baseURL url.URL
}

func NewInfoV2(baseURL url.URL) *InfoV2 {
	return &InfoV2{
		baseURL: baseURL,
	}
}

func (h *InfoV2) UnauthenticatedRoutes() []routing.Route {
	return nil
}

func (h *InfoV2) get(r *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForV2Info()), nil
}

func (h *InfoV2) AuthenticatedRoutes() []routing.Route {
	return []routing.Route{
		{Method: "GET", Pattern: V2InfoPath, Handler: h.get},
	}
}
