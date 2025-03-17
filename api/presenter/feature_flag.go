package presenter

import (
	"net/url"
	"time"

	"code.cloudfoundry.org/korifi/tools"
)

const featureFlagBase = "/v3/feature_flags"

type FeatureFlagResponse struct {
	Name          string           `json:"name"`
	Enabled       bool             `json:"enabled"`
	UpdatedAt     string           `json:"updated_at"`
	CustomMessage string           `json:"custom_error_message"`
	Links         FeatureFlagLinks `json:"links"`
}

type FeatureFlagsResponse struct {
	PaginationData PaginationData        `json:"pagination"`
	Resources      []FeatureFlagResponse `json:"resources"`
}

type FeatureFlagLinks struct {
	Self Link `json:"self"`
}

func ForFeatureFlag(baseURL url.URL) FeatureFlagResponse {
	return FeatureFlagResponse{
		Name:          "diego_docker",
		Enabled:       false,
		UpdatedAt:     tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
		CustomMessage: "When enabled, Docker applications are supported by Diego. When disabled, Docker applications will stop running. It will still be possible to stop and delete them and update their configurations.",
		Links: FeatureFlagLinks{
			Self: Link{
				HRef: buildURL(baseURL).appendPath(featureFlagBase, "diego_docker").build(),
			},
		},
	}
}

func ForFeatureFlags(baseURL url.URL) FeatureFlagsResponse {
	return FeatureFlagsResponse{
		PaginationData: PaginationData{
			TotalResults: 1,
			TotalPages:   1,
			First: PageRef{
				HREF: buildURL(baseURL).appendPath(featureFlagBase, "diego_docker").build(),
			},
			Last: PageRef{
				HREF: buildURL(baseURL).appendPath(featureFlagBase, "diego_docker").build(),
			},
		},
		Resources: []FeatureFlagResponse{
			{
				Name:          "diego_docker",
				Enabled:       false,
				UpdatedAt:     tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
				CustomMessage: "When enabled, Docker applications are supported by Diego. When disabled, Docker applications will stop running. It will still be possible to stop and delete them and update their configurations.",
				Links: FeatureFlagLinks{
					Self: Link{
						HRef: buildURL(baseURL).appendPath(featureFlagBase, "diego_docker").build(),
					},
				},
			},
		},
	}
}
