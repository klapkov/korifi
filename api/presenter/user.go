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

type UsersResponse struct {
	PaginationData PaginationData `json:"pagination"`
	Resources      []UserResponse `json:"resources"`
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

func ForUsers(baseURL url.URL) UsersResponse {
	return UsersResponse{
		PaginationData: PaginationData{
			TotalResults: 1,
			TotalPages:   1,
			First: PageRef{
				HREF: buildURL(baseURL).appendPath(usersBase, "mock-user-1").build(),
			},
			Last: PageRef{
				HREF: buildURL(baseURL).appendPath(usersBase, "mock-user-3").build(),
			},
		},
		Resources: []UserResponse{
			{
				GUID:             "mock-user-1",
				CreatedAt:        tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
				UpdatedAt:        tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
				Name:             "mock-user-1",
				PresentationName: "mock-user-1",
				Origin:           "uaa",
				Links: UserLinks{
					Self: Link{
						HRef: buildURL(baseURL).appendPath(usersBase, "mock-user-1").build(),
					},
				},
			},
			{
				GUID:             "mock-user-2",
				CreatedAt:        tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
				UpdatedAt:        tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
				Name:             "mock-user-2",
				PresentationName: "mock-user-2",
				Origin:           "uaa",
				Links: UserLinks{
					Self: Link{
						HRef: buildURL(baseURL).appendPath(usersBase, "mock-user-2").build(),
					},
				},
			},
			{
				GUID:             "mock-user-3",
				CreatedAt:        tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
				UpdatedAt:        tools.ZeroIfNil(formatTimestamp(tools.PtrTo(time.Now()))),
				Name:             "mock-user-3",
				PresentationName: "mock-user-3",
				Origin:           "uaa",
				Links: UserLinks{
					Self: Link{
						HRef: buildURL(baseURL).appendPath(usersBase, "mock-user-3").build(),
					},
				},
			},
		},
	}
}
