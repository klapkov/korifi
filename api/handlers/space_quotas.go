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
	SpaceQuotasPath      = "/v3/space_quotas"
	SpaceQuotaPath       = "/v3/space_quotas/{guid}"
	ApplySpaceQuotaPath  = "/v3/space_quotas/{guid}/relationships/spaces"
	RemoveSpaceQuotaPath = "/v3/space_quotas/{guid}/relationships/spaces/{space_guid}"
)

type SpaceQuotas struct {
	apiBaseURL                               url.URL
	orgRepo                                  CFOrgRepository
	domainRepo                               CFDomainRepository
	requestValidator                         RequestValidator
	userCertificateExpirationWarningDuration time.Duration
	defaultDomainName                        string
}

func NewSpaceQuotas(apiBaseURL url.URL, requestValidator RequestValidator) *SpaceQuotas {
	return &SpaceQuotas{
		apiBaseURL:       apiBaseURL,
		requestValidator: requestValidator,
	}
}

func (h *SpaceQuotas) create(r *http.Request) (*routing.Response, error) {
	// authInfo, _ := authorization.InfoFromContext(r.Context())
	// logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.space-quotas.create")
	quota := repositories.SpaceQuotaRecord{
		GUID:      uuid.NewString(),
		Name:      "space-quota-1",
		CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
	}
	return routing.NewResponse(http.StatusCreated).WithBody(presenter.ForSpaceQuota(quota, h.apiBaseURL)), nil
}

func (h *SpaceQuotas) get(r *http.Request) (*routing.Response, error) {
	// authInfo, _ := authorization.InfoFromContext(r.Context())
	// logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.space-quotas.get")
	orgQuotaGUID := routing.URLParam(r, "guid")
	orgQuotas := repositories.SpaceQuotaRecord{
		GUID:      orgQuotaGUID,
		Name:      "org-quota-name-1",
		CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForSpaceQuota(orgQuotas, h.apiBaseURL)), nil
}

func (h *SpaceQuotas) list(r *http.Request) (*routing.Response, error) {
	// authInfo, _ := authorization.InfoFromContext(r.Context())
	// logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.space-quotas.list")

	orgQuotas := []repositories.SpaceQuotaRecord{
		{
			GUID:      uuid.NewString(),
			Name:      "org-quota-name-1",
			CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			GUID:      uuid.NewString(),
			Name:      "org-quota-name-2",
			CreatedAt: time.Date(2025, time.January, 5, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, time.February, 5, 12, 0, 0, 0, time.UTC),
		},
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForList(presenter.ForSpaceQuota, orgQuotas, h.apiBaseURL, *r.URL)), nil
}

func (h *SpaceQuotas) update(r *http.Request) (*routing.Response, error) {
	// authInfo, _ := authorization.InfoFromContext(r.Context())
	// logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.space-quotas.update")

	orgQuotaGUID := routing.URLParam(r, "guid")
	orgQuotas := repositories.SpaceQuotaRecord{
		GUID:      orgQuotaGUID,
		Name:      "org-quota-name-1",
		CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForSpaceQuota(orgQuotas, h.apiBaseURL)), nil
}

func (h *SpaceQuotas) delete(r *http.Request) (*routing.Response, error) {
	// authInfo, _ := authorization.InfoFromContext(r.Context())
	// logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.space-quotas.delete")

	quotaGUID := routing.URLParam(r, "guid")

	return routing.NewResponse(http.StatusAccepted).WithHeader("Location", presenter.JobURLForRedirects(quotaGUID, presenter.OrgDeleteOperation, h.apiBaseURL)), nil
}

func (h *SpaceQuotas) applyQuota(r *http.Request) (*routing.Response, error) {
	// authInfo, _ := authorization.InfoFromContext(r.Context())
	// logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.space-quotas.delete")

	spaceQuotas := []repositories.SpaceData{
		{GUID: "org-guid-1"},
		{GUID: "org-guid-2"},
		{GUID: "org-guid-3"},
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForQuotaToSpaces(spaceQuotas, h.apiBaseURL)), nil
}

func (h *SpaceQuotas) removeQuota(r *http.Request) (*routing.Response, error) {
	// authInfo, _ := authorization.InfoFromContext(r.Context())
	// logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.space-quotas.delete")

	return routing.NewResponse(http.StatusNoContent), nil
}

func (h *SpaceQuotas) UnauthenticatedRoutes() []routing.Route {
	return nil
}

func (h *SpaceQuotas) AuthenticatedRoutes() []routing.Route {
	return []routing.Route{
		{Method: "POST", Pattern: SpaceQuotasPath, Handler: h.create},
		{Method: "GET", Pattern: SpaceQuotaPath, Handler: h.get},
		{Method: "GET", Pattern: SpaceQuotasPath, Handler: h.list},
		{Method: "PATCH", Pattern: SpaceQuotaPath, Handler: h.update},
		{Method: "DELETE", Pattern: SpaceQuotaPath, Handler: h.delete},
		{Method: "POST", Pattern: ApplySpaceQuotaPath, Handler: h.applyQuota},
		{Method: "DELETE", Pattern: RemoveSpaceQuotaPath, Handler: h.removeQuota},
	}
}
