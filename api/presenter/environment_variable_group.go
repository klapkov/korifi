package presenter

import (
	"net/url"
	"time"

	"code.cloudfoundry.org/korifi/tools"
)

const envVarGroupBase = "/v3/environment_variable_groups"

type EnvVarGroupResponse struct {
	UpdatedAt string           `json:"updated_at"`
	Name      string           `json:"name"`
	Var       map[string]any   `json:"var"`
	Links     FeatureFlagLinks `json:"links"`
}

type EnvVarGroupsResponse struct {
	PaginationData PaginationData        `json:"pagination"`
	Resources      []EnvVarGroupResponse `json:"resources"`
}

type EnvVarGroupLinks struct {
	Self Link `json:"self"`
}

func ForEnvVarGroup(baseURL url.URL, group string) EnvVarGroupResponse {
	return EnvVarGroupResponse{
		UpdatedAt: tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
		Name:      group,
		Var: map[string]any{
			"foo": "bar",
		},
		Links: FeatureFlagLinks{
			Self: Link{
				HRef: buildURL(baseURL).appendPath(envVarGroupBase, group).build(),
			},
		},
	}
}

func ForEnvVarGroups(baseURL url.URL) EnvVarGroupsResponse {
	return EnvVarGroupsResponse{
		PaginationData: PaginationData{
			TotalResults: 1,
			TotalPages:   1,
			First: PageRef{
				HREF: buildURL(baseURL).appendPath(envVarGroupBase, "running").build(),
			},
			Last: PageRef{
				HREF: buildURL(baseURL).appendPath(envVarGroupBase, "staging").build(),
			},
		},
		Resources: []EnvVarGroupResponse{
			{
				UpdatedAt: tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
				Name:      "running",
				Var: map[string]any{
					"foo": "bar",
				},
				Links: FeatureFlagLinks{
					Self: Link{
						HRef: buildURL(baseURL).appendPath(envVarGroupBase, "running").build(),
					},
				},
			},
			{
				UpdatedAt: tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
				Name:      "staging",
				Var: map[string]any{
					"foo": "bar",
				},
				Links: FeatureFlagLinks{
					Self: Link{
						HRef: buildURL(baseURL).appendPath(envVarGroupBase, "staging").build(),
					},
				},
			},
		},
	}
}
