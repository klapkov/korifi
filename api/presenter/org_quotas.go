package presenter

import (
	"net/url"

	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/api/repositories/include"
)

type OrgQuotasResponse struct {
	Name      string   `json:"name"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	GUID      string   `json:"guid"`
	Links     OrgLinks `json:"links"`
}

type QuotaToOrgsResponse struct {
	Data  []QuotaToOrgs `json:"data"`
	Links OrgLinks      `json:"links"`
}

type QuotaToOrgs struct {
	GUID string `json:"guid"`
}

func ForOrgQuota(org repositories.OrgQuotaRecord, apiBaseURL url.URL, includes ...include.Resource) OrgQuotasResponse {
	return OrgQuotasResponse{
		Name:      org.Name,
		GUID:      org.GUID,
		CreatedAt: org.CreatedAt.String(),
		UpdatedAt: org.CreatedAt.String(),
		Links: OrgLinks{
			Self: &Link{
				HRef: buildURL(apiBaseURL).appendPath("/v3/organization_quotas", org.GUID).build(),
			},
		},
	}
}

func ForQuotaToOrgs(orgs []repositories.OrgData, apiBaseURL url.URL, includes ...include.Resource) QuotaToOrgsResponse {
	resp := QuotaToOrgsResponse{
		Data: []QuotaToOrgs{},
		Links: OrgLinks{
			Self: &Link{
				HRef: buildURL(apiBaseURL).appendPath("/v3/organization_quotas", "quota-org-guid", "relationships/organizations").build(),
			},
		},
	}

	for _, org := range orgs {
		resp.Data = append(resp.Data, QuotaToOrgs{GUID: org.GUID})
	}

	return resp
}
