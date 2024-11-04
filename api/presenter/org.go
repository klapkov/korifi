package presenter

import (
	"net/url"

	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/model"
)

const (
	// TODO: repetition with handler endpoint?
	orgsBase = "/v3/organizations"
)

type OrgResponse struct {
	Name string `json:"name"`
	GUID string `json:"guid"`

	CreatedAt     string                             `json:"created_at"`
	UpdatedAt     string                             `json:"updated_at"`
	Suspended     bool                               `json:"suspended"`
	Relationships map[string]model.ToOneRelationship `json:"relationships,omitempty"`
	Metadata      Metadata                           `json:"metadata"`
	Links         OrgLinks                           `json:"links"`
}

type OrgLinks struct {
	Self          *Link `json:"self"`
	Domains       *Link `json:"domains,omitempty"`
	DefaultDomain *Link `json:"default_domain,omitempty"`
	Quota         *Link `json:"quota,omitempty"`
}

func ForOrg(org repositories.OrgRecord, apiBaseURL url.URL, includes ...model.IncludedResource) OrgResponse {
	return OrgResponse{
		Name:      org.Name,
		GUID:      org.GUID,
		CreatedAt: formatTimestamp(&org.CreatedAt),
		UpdatedAt: formatTimestamp(org.UpdatedAt),
		Suspended: org.Suspended,
		Metadata: Metadata{
			Labels:      emptyMapIfNil(org.Labels),
			Annotations: emptyMapIfNil(org.Annotations),
		},
		Links: OrgLinks{
			Self: &Link{
				HRef: buildURL(apiBaseURL).appendPath(orgsBase, org.GUID).build(),
			},
		},
	}
}
