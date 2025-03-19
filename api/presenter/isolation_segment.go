package presenter

import (
	"net/url"

	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/api/repositories/include"
)

type IsolationSegmentResponse struct {
	Name      string                `json:"name"`
	CreatedAt string                `json:"created_at"`
	UpdatedAt string                `json:"updated_at"`
	GUID      string                `json:"guid"`
	Links     IsolationSegmentLinks `json:"links"`
}

type IsolationSegmentOrgsResponse struct {
	Data  []QuotaToOrgs `json:"data"`
	Links OrgLinks      `json:"links"`
}

type IsolationSegmentLinks struct {
	Self         *Link `json:"self"`
	Organization *Link `json:"organization"`
}

func ForIsolationSegment(segm repositories.IsolationSegmentRecord, apiBaseURL url.URL, includes ...include.Resource) IsolationSegmentResponse {
	return IsolationSegmentResponse{
		Name:      segm.Name,
		GUID:      segm.GUID,
		CreatedAt: segm.CreatedAt.String(),
		UpdatedAt: segm.CreatedAt.String(),
		Links: IsolationSegmentLinks{
			Self: &Link{
				HRef: buildURL(apiBaseURL).appendPath("/v3/isolation_segments", segm.GUID).build(),
			},
		},
	}
}

func ForIsolationSegmentToOrgs(orgs []repositories.OrgData, apiBaseURL url.URL, includes ...include.Resource) IsolationSegmentOrgsResponse {
	resp := IsolationSegmentOrgsResponse{
		Data: []QuotaToOrgs{},
		Links: OrgLinks{
			Self: &Link{
				HRef: buildURL(apiBaseURL).appendPath("/v3/isolation_segments", "quota-org-guid", "relationships/organizations").build(),
			},
		},
	}

	for _, org := range orgs {
		resp.Data = append(resp.Data, QuotaToOrgs{GUID: org.GUID})
	}

	return resp
}
