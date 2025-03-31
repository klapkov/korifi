package presenter

import (
	"net/url"
	"slices"

	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/api/repositories/include"
	"code.cloudfoundry.org/korifi/tools"
	"github.com/BooleanCat/go-functional/v2/it"
	"github.com/BooleanCat/go-functional/v2/it/itx"
)

const (
	// TODO: repetition with handler endpoint?
	spacesBase = "/v3/spaces"
)

type SpaceResponse struct {
	Name          string                       `json:"name"`
	GUID          string                       `json:"guid"`
	CreatedAt     string                       `json:"created_at"`
	UpdatedAt     string                       `json:"updated_at"`
	Links         SpaceLinks                   `json:"links"`
	Metadata      Metadata                     `json:"metadata"`
	Relationships map[string]ToOneRelationship `json:"relationships"`
}

type SpaceLinks struct {
	Self         *Link `json:"self"`
	Organization *Link `json:"organization"`
}

func ForSpace(space repositories.SpaceRecord, apiBaseURL url.URL, includes ...include.Resource) SpaceResponse {
	return SpaceResponse{
		Name:      space.Name,
		GUID:      space.GUID,
		CreatedAt: tools.ZeroIfNil(formatTimestamp(&space.CreatedAt)),
		UpdatedAt: tools.ZeroIfNil(formatTimestamp(space.UpdatedAt)),
		Metadata: Metadata{
			Labels:      emptyMapIfNil(space.Labels),
			Annotations: emptyMapIfNil(space.Annotations),
		},
		Relationships: ForRelationships(space.Relationships()),
		Links: SpaceLinks{
			Self: &Link{
				HRef: buildURL(apiBaseURL).appendPath(spacesBase, space.GUID).build(),
			},
			Organization: &Link{
				HRef: buildURL(apiBaseURL).appendPath(orgsBase, space.OrganizationGUID).build(),
			},
		},
	}
}

func ForSpaceList(spaceRecords []repositories.SpaceRecord, orgRecords []repositories.OrgRecord, baseURL, requestURL url.URL) ListResponse[SpaceResponse] {
	includedOrgs := slices.Collect(it.Map(itx.FromSlice(orgRecords), func(org repositories.OrgRecord) include.Resource {
		return include.Resource{
			Type:     "organizations",
			Resource: ForOrg(org, baseURL),
		}
	}))

	return ForList(ForSpace, spaceRecords, baseURL, requestURL, includedOrgs...)
}
