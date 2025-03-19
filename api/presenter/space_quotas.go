package presenter

import (
	"net/url"

	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/api/repositories/include"
)

type SpaceQuotasResponse struct {
	Name      string     `json:"name"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
	GUID      string     `json:"guid"`
	Links     SpaceLinks `json:"links"`
}

type QuotaToSpaceResponse struct {
	Data  []QuotaToSpace `json:"data"`
	Links SpaceLinks     `json:"links"`
}

type QuotaToSpace struct {
	GUID string `json:"guid"`
}

func ForSpaceQuota(space repositories.SpaceQuotaRecord, apiBaseURL url.URL, includes ...include.Resource) SpaceQuotasResponse {
	return SpaceQuotasResponse{
		Name:      space.Name,
		GUID:      space.GUID,
		CreatedAt: space.CreatedAt.String(),
		UpdatedAt: space.CreatedAt.String(),
		Links: SpaceLinks{
			Self: &Link{
				HRef: buildURL(apiBaseURL).appendPath("/v3/space_quotas", space.GUID).build(),
			},
		},
	}
}

func ForQuotaToSpaces(spaces []repositories.SpaceData, apiBaseURL url.URL, includes ...include.Resource) QuotaToSpaceResponse {
	resp := QuotaToSpaceResponse{
		Data: []QuotaToSpace{},
		Links: SpaceLinks{
			Self: &Link{
				HRef: buildURL(apiBaseURL).appendPath("/v3/space_quotas", "quota-space-guid", "relationships/organizations").build(),
			},
		},
	}

	for _, space := range spaces {
		resp.Data = append(resp.Data, QuotaToSpace{GUID: space.GUID})
	}

	return resp
}
