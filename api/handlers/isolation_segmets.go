package handlers

import (
	"net/http"
	"net/url"
	"time"

	"code.cloudfoundry.org/korifi/api/presenter"
	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/api/routing"
	"github.com/google/uuid"
)

const (
	IsolationSegmentsPath      = "/v3/isolation_segments"
	IsolationSegmentPath       = "/v3/isolation_segments/{guid}"
	IsolationSegmentOrgsPath   = "/v3/isolation_segments/:guid/relationships/organizations"
	IsolationSegmentOrgPath    = "/v3/isolation_segments/:guid/relationships/organizations/:org_guid"
	IsolationSegmentSpacesPath = "/v3/isolation_segments/:guid/relationships/spaces"
)

type IsolationSegments struct {
	apiBaseURL       url.URL
	requestValidator RequestValidator
}

func NewIsolationSegments(apiBaseURL url.URL, requestValidator RequestValidator) *IsolationSegments {
	return &IsolationSegments{
		apiBaseURL:       apiBaseURL,
		requestValidator: requestValidator,
	}
}

func (h *IsolationSegments) create(r *http.Request) (*routing.Response, error) {
	//authInfo, _ := authorization.InfoFromContext(r.Context())
	//logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.org_quotas.get")

	isolSegment := repositories.IsolationSegmentRecord{
		GUID:      uuid.NewString(),
		Name:      "isolation-segment-name-1",
		CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForIsolationSegment(isolSegment, h.apiBaseURL)), nil
}

func (h *IsolationSegments) get(r *http.Request) (*routing.Response, error) {
	//authInfo, _ := authorization.InfoFromContext(r.Context())
	//logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.org_quotas.get")

	isolSegmentGUID := routing.URLParam(r, "guid")
	isolSegment := repositories.IsolationSegmentRecord{
		GUID:      isolSegmentGUID,
		Name:      "isolation-segment-name-1",
		CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForIsolationSegment(isolSegment, h.apiBaseURL)), nil
}

func (h *IsolationSegments) list(r *http.Request) (*routing.Response, error) {
	//authInfo, _ := authorization.InfoFromContext(r.Context())
	//logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.org_quotas.get")

	isolSegment := []repositories.IsolationSegmentRecord{
		{
			GUID:      uuid.NewString(),
			Name:      "is-name-1",
			CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			GUID:      uuid.NewString(),
			Name:      "is-name-2",
			CreatedAt: time.Date(2025, time.January, 5, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, time.February, 5, 12, 0, 0, 0, time.UTC),
		},
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForList(presenter.ForIsolationSegment, isolSegment, h.apiBaseURL, *r.URL)), nil
}

func (h *IsolationSegments) listOrgs(r *http.Request) (*routing.Response, error) {
	orgs := []repositories.OrgData{
		{GUID: "org-guid-1"},
		{GUID: "org-guid-2"},
		{GUID: "org-guid-3"},
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForIsolationSegmentToOrgs(orgs, h.apiBaseURL)), nil
}

func (h *IsolationSegments) listSpaces(r *http.Request) (*routing.Response, error) {
	orgs := []repositories.OrgData{
		{GUID: "org-guid-1"},
		{GUID: "org-guid-2"},
		{GUID: "org-guid-3"},
	}
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForIsolationSegmentToOrgs(orgs, h.apiBaseURL)), nil
}

func (h *IsolationSegments) update(r *http.Request) (*routing.Response, error) {
	isolSegmentGUID := routing.URLParam(r, "guid")
	isolSegment := repositories.IsolationSegmentRecord{
		GUID:      isolSegmentGUID,
		Name:      "isolation-segment-name-1",
		CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForIsolationSegment(isolSegment, h.apiBaseURL)), nil
}

func (h *IsolationSegments) delete(r *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusNoContent), nil
}

func (h *IsolationSegments) addOrgs(r *http.Request) (*routing.Response, error) {
	orgs := []repositories.OrgData{
		{GUID: "org-guid-1"},
		{GUID: "org-guid-2"},
		{GUID: "org-guid-3"},
	}
	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForIsolationSegmentToOrgs(orgs, h.apiBaseURL)), nil
}

func (h *IsolationSegments) removeOrg(r *http.Request) (*routing.Response, error) {
	return routing.NewResponse(http.StatusNoContent), nil
}

func (h *IsolationSegments) UnauthenticatedRoutes() []routing.Route {
	return nil
}

func (h *IsolationSegments) AuthenticatedRoutes() []routing.Route {
	return []routing.Route{
		{Method: "POST", Pattern: IsolationSegmentsPath, Handler: h.create},
		{Method: "GET", Pattern: IsolationSegmentsPath, Handler: h.list},
		{Method: "GET", Pattern: IsolationSegmentPath, Handler: h.get},
		{Method: "GET", Pattern: IsolationSegmentOrgsPath, Handler: h.listOrgs},
		{Method: "GET", Pattern: IsolationSegmentSpacesPath, Handler: h.listSpaces},
		{Method: "PATCH", Pattern: IsolationSegmentPath, Handler: h.update},
		{Method: "DELETE", Pattern: IsolationSegmentPath, Handler: h.delete},
		{Method: "POST", Pattern: IsolationSegmentOrgsPath, Handler: h.addOrgs},
		{Method: "DELETE", Pattern: IsolationSegmentOrgPath, Handler: h.removeOrg},
	}
}
