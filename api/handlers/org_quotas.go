package handlers

import (
	"net/http"
	"net/url"
	"time"

	apierrors "code.cloudfoundry.org/korifi/api/errors"
	"code.cloudfoundry.org/korifi/api/payloads"
	"code.cloudfoundry.org/korifi/api/presenter"
	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/api/routing"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

const (
	OrgQuotasPath   = "/v3/organization_quotas"
	OrgQuotaPath    = "/v3/organization_quotas/{guid}"
	OrgQuotaRelPath = "/v3/organization_quotas/{guid}/relationships/organizations"
)

type OrgQuotas struct {
	apiBaseURL       url.URL
	requestValidator RequestValidator
}

func NewOrgQuotas(apiBaseURL url.URL, requestValidator RequestValidator) *OrgQuotas {
	return &OrgQuotas{
		apiBaseURL:       apiBaseURL,
		requestValidator: requestValidator,
	}
}

func (h *OrgQuotas) get(r *http.Request) (*routing.Response, error) {
	//authInfo, _ := authorization.InfoFromContext(r.Context())
	//logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.org_quotas.get")

	orgQuotaGUID := routing.URLParam(r, "guid")
	orgQuotas := repositories.OrgQuotaRecord{
		GUID:      orgQuotaGUID,
		Name:      "org-quota-name-1",
		CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForOrgQuota(orgQuotas, h.apiBaseURL)), nil
}

func (h *OrgQuotas) create(r *http.Request) (*routing.Response, error) {
	//authInfo, _ := authorization.InfoFromContext(r.Context())
	//logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.org.create")

	orgQuotas := repositories.OrgQuotaRecord{
		GUID:      uuid.NewString(),
		Name:      "org-quota-name-1",
		CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForOrgQuota(orgQuotas, h.apiBaseURL)), nil
}

func (h *OrgQuotas) list(r *http.Request) (*routing.Response, error) {
	var orgQuotas []repositories.OrgQuotaRecord
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.org-quotas.list")

	payload := new(payloads.OrgQuotasList)
	if err := h.requestValidator.DecodeAndValidateJSONPayload(r, payload); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to decode payload")
	}

	if payload.Names != "" {
		orgQuotas = append(orgQuotas, repositories.OrgQuotaRecord{
			GUID:      uuid.NewString(),
			Name:      payload.Names,
			CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
		})
	} else {
		orgQuotas = []repositories.OrgQuotaRecord{
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
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForList(presenter.ForOrgQuota, orgQuotas, h.apiBaseURL, *r.URL)), nil
}

func (h *OrgQuotas) delete(r *http.Request) (*routing.Response, error) {
	//authInfo, _ := authorization.InfoFromContext(r.Context())
	//logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.org-quotas.delete")

	orgQuotaGUID := routing.URLParam(r, "guid")
	return routing.NewResponse(http.StatusAccepted).WithHeader("Location", presenter.JobURLForRedirects(orgQuotaGUID, presenter.OrgDeleteOperation, h.apiBaseURL)), nil
}

func (h *OrgQuotas) update(r *http.Request) (*routing.Response, error) {
	//authInfo, _ := authorization.InfoFromContext(r.Context())
	//logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.org-quotas.update")

	orgQuotaGUID := routing.URLParam(r, "guid")
	orgQuotas := repositories.OrgQuotaRecord{
		GUID:      orgQuotaGUID,
		Name:      "org-quota-name-1",
		CreatedAt: time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, time.February, 1, 12, 0, 0, 0, time.UTC),
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForOrgQuota(orgQuotas, h.apiBaseURL)), nil
}

func (h *OrgQuotas) applyQuota(r *http.Request) (*routing.Response, error) {
	//authInfo, _ := authorization.InfoFromContext(r.Context())
	//logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.org.create")

	orgQuotas := []repositories.OrgData{
		{GUID: "org-guid-1"},
		{GUID: "org-guid-2"},
		{GUID: "org-guid-3"},
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForQuotaToOrgs(orgQuotas, h.apiBaseURL)), nil
}

func (h *OrgQuotas) UnauthenticatedRoutes() []routing.Route {
	return nil
}

func (h *OrgQuotas) AuthenticatedRoutes() []routing.Route {
	return []routing.Route{
		{Method: "GET", Pattern: OrgQuotaPath, Handler: h.get},
		{Method: "POST", Pattern: OrgQuotasPath, Handler: h.create},
		{Method: "POST", Pattern: OrgQuotaRelPath, Handler: h.applyQuota},
		{Method: "GET", Pattern: OrgQuotasPath, Handler: h.list},
		{Method: "DELETE", Pattern: OrgQuotaPath, Handler: h.delete},
		{Method: "PATCH", Pattern: OrgQuotaPath, Handler: h.update},
	}
}
