package presenter

import (
	"net/url"
	"time"

	"code.cloudfoundry.org/korifi/api/repositories/include"
	"code.cloudfoundry.org/korifi/tools"
)

const usersBase = "/v3/users"

type UserResponse struct {
	GUID             string    `json:"guid"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
	Name             string    `json:"username"`
	PresentationName string    `json:"presentation_name"`
	Origin           string    `json:"origin"`
	Links            UserLinks `json:"links"`
}

type UserLinks struct {
	Self Link `json:"self"`
}

func ForUser(name string, baseURL url.URL, includes ...include.Resource) UserResponse {
	return UserResponse{
		GUID:             name,
		CreatedAt:        tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
		UpdatedAt:        tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
		Name:             name,
		PresentationName: name,
		Origin:           "uaa",
		Links: UserLinks{
			Self: Link{
				HRef: buildURL(baseURL).appendPath(usersBase, name).build(),
			},
		},
	}
}
