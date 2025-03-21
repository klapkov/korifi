package handlers

import (
	"net/http"
	"net/url"

	"code.cloudfoundry.org/korifi/api/presenter"
	"code.cloudfoundry.org/korifi/api/routing"
)

const (
	usersPath = "/v3/users"
	userPath  = "/v3/users/{guid}"
)

type User struct {
	apiBaseURL url.URL
}

func NewUser(apiBaseURL url.URL) User {
	return User{
		apiBaseURL: apiBaseURL,
	}
}

func (h User) create(req *http.Request) (*routing.Response, error) {
	user := routing.URLParam(req, "guid")
	return routing.NewResponse(http.StatusCreated).WithBody(presenter.ForUser(user, h.apiBaseURL)), nil
}

func (h User) get(req *http.Request) (*routing.Response, error) {
	user := routing.URLParam(req, "guid")
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForUser(user, h.apiBaseURL)), nil
}

func (h User) update(req *http.Request) (*routing.Response, error) {
	user := routing.URLParam(req, "guid")
	return routing.NewResponse(http.StatusCreated).WithBody(presenter.ForUser(user, h.apiBaseURL)), nil
}

func (h User) list(req *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForUsers(h.apiBaseURL)), nil
}

func (h User) delete(req *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusAccepted), nil
}

func (h User) UnauthenticatedRoutes() []routing.Route {
	return nil
}

func (h User) AuthenticatedRoutes() []routing.Route {
	return []routing.Route{
		{Method: "GET", Pattern: usersPath, Handler: h.list},
		{Method: "GET", Pattern: userPath, Handler: h.get},
		{Method: "PATCH", Pattern: userPath, Handler: h.update},
		{Method: "POST", Pattern: usersPath, Handler: h.create},
		{Method: "DELETE", Pattern: userPath, Handler: h.delete},
	}
}
