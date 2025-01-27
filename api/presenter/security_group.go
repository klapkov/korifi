package presenter

import (
	"net/url"
	"slices"

	"code.cloudfoundry.org/korifi/api/payloads"
	"code.cloudfoundry.org/korifi/api/repositories"
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/model"
	"github.com/BooleanCat/go-functional/v2/it"
)

const securityGroupBase = "/v3/security_groups"

type SecurityGroupResponse struct {
	GUID            string                             `json:"guid"`
	CreatedAt       string                             `json:"created_at"`
	Name            string                             `json:"name"`
	GloballyEnabled korifiv1alpha1.GloballyEnabled     `json;"globally_enabled"`
	Rules           []korifiv1alpha1.SecurityGroupRule `json:"rules"`
	Links           SecurityGroupLinks                 `json:"links"`
}

type SecurityGroupRunningSpacesResponse struct {
	Data  []payloads.RelationshipData `json:"data"`
	Links SecurityGroupLinks          `json:"links"`
}

type SecurityGroupLinks struct {
	Self Link `json:"self"`
}

func ForSecurityGroup(securityGroupRecord repositories.SecurityGroupRecord, baseURL url.URL, includes ...model.IncludedResource) SecurityGroupResponse {
	return SecurityGroupResponse{
		GUID:            securityGroupRecord.GUID,
		CreatedAt:       formatTimestamp(&securityGroupRecord.CreatedAt),
		Name:            securityGroupRecord.Name,
		GloballyEnabled: securityGroupRecord.GloballyEnabled,
		Rules:           securityGroupRecord.Rules,
		Links: SecurityGroupLinks{
			Self: Link{
				HRef: buildURL(baseURL).appendPath(securityGroupBase, securityGroupRecord.GUID).build(),
			},
		},
	}
}

func ForSecurityGroupRunningSpaces(securityGroupRecord repositories.SecurityGroupRecord, baseURL url.URL) SecurityGroupRunningSpacesResponse {
	return SecurityGroupRunningSpacesResponse{
		Data: slices.Collect(it.Map(slices.Values(securityGroupRecord.RunningSpaces), func(v string) payloads.RelationshipData {
			return payloads.RelationshipData{GUID: v}
		})),
		Links: SecurityGroupLinks{
			Self: Link{
				HRef: buildURL(baseURL).appendPath(securityGroupBase, securityGroupRecord.GUID, "relationships", "running_spaces").build(),
			},
		},
	}
}
