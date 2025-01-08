package payloads

import (
	"net/url"
	"slices"

	"code.cloudfoundry.org/korifi/api/payloads/parse"
	"code.cloudfoundry.org/korifi/api/repositories"
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"github.com/BooleanCat/go-functional/v2/it"
	jellidation "github.com/jellydator/validation"
)

type SecurityGroupCreate struct {
	Name            string                             `json:"name"`
	Rules           []korifiv1alpha1.SecurityGroupRule `json:"rules"`
	GloballyEnabled korifiv1alpha1.GloballyEnabled     `json:"globally_enabled"`
}

func (c SecurityGroupCreate) Validate() error {
	return jellidation.ValidateStruct(&c,
		jellidation.Field(&c.Name, jellidation.Required),
		jellidation.Field(&c.Rules, jellidation.Required),
	)
}

func (c SecurityGroupCreate) ToMessage() repositories.CreateSecurityGroupMessage {
	return repositories.CreateSecurityGroupMessage{
		DisplayName:     c.Name,
		Rules:           c.Rules,
		GloballyEnabled: c.GloballyEnabled,
	}
}

type SecurityGroupList struct {
	GUIDs                  string `json:"guids"`
	Names                  string `json:"names"`
	GloballyEnabledStaging bool   `json:"globally_enabled_staging"`
	GloballyEnabledRunning bool   `json:"globally_enabled_running"`
	RunningSpaceGUIDs      string `json:"running_space_guids"`
	StagingSpaceGUIDs      string `json:"staging_space_guids"`
}

func (l SecurityGroupList) SupportedKeys() []string {
	return []string{
		"guids",
		"names",
		"globally_enabled_staging",
		"globally_enabled_running",
		"running_space_guids",
		"staging_space_guids",
	}
}

func (l *SecurityGroupList) DecodeFromURLValues(values url.Values) error {
	var err error

	l.GUIDs = values.Get("guids")
	l.Names = values.Get("names")
	if l.GloballyEnabledStaging, err = getBool(values, "globally_enabled_staging"); err != nil {
		return err
	}
	if l.GloballyEnabledRunning, err = getBool(values, "globally_enabled_running"); err != nil {
		return err
	}
	l.RunningSpaceGUIDs = values.Get("running_space_guids")
	l.StagingSpaceGUIDs = values.Get("staging_space_guids")

	return nil
}

func (l SecurityGroupList) ToMessage() repositories.ListSecurityGroupMessage {
	return repositories.ListSecurityGroupMessage{
		GUIDs:                  parse.ArrayParam(l.GUIDs),
		Names:                  parse.ArrayParam(l.Names),
		GloballyEnabledStaging: &l.GloballyEnabledStaging,
		GloballyEnabledRunning: &l.GloballyEnabledRunning,
		RunningSpaceGUIDs:      parse.ArrayParam(l.RunningSpaceGUIDs),
		StagingSpaceGUIDs:      parse.ArrayParam(l.StagingSpaceGUIDs),
	}
}

// func (l *SecurityGroupCreate) SupportedKeys() []string {
// 	return []string{"name", "rules"}
// }

type SecurityGroupUpdate struct {
	DisplayName     string                               `json:"name"`
	GloballyEnabled korifiv1alpha1.GloballyEnabledUpdate `json:"globally_enabled"`
	Rules           []korifiv1alpha1.SecurityGroupRule   `json:"rules"`
}

func (u SecurityGroupUpdate) ToMessage(guid string) repositories.UpdateSecurityGroupMessage {
	return repositories.UpdateSecurityGroupMessage{
		GUID:            guid,
		DisplayName:     u.DisplayName,
		GloballyEnabled: u.GloballyEnabled,
		Rules:           u.Rules,
	}
}

type SecurityGroupBindRunning struct {
	Data []RelationshipData `json:"data"`
}

func (b SecurityGroupBindRunning) Validate() error {
	return jellidation.ValidateStruct(&b,
		jellidation.Field(&b.Data, jellidation.Required),
	)
}

func (b SecurityGroupBindRunning) ToMessage(guid string) repositories.BindRunningSecurityGroupMessage {
	return repositories.BindRunningSecurityGroupMessage{
		GUID: guid,
		Spaces: slices.Collect(it.Map(slices.Values(b.Data), func(v RelationshipData) string {
			return v.GUID
		})),
	}
}

type SecurityGroupBindStaging struct {
	Data []RelationshipData `json:"data"`
}

func (b SecurityGroupBindStaging) Validate() error {
	return jellidation.ValidateStruct(&b,
		jellidation.Field(&b.Data, jellidation.Required),
	)
}

func (b SecurityGroupBindStaging) ToMessage(guid string) repositories.BindStagingSecurityGroupMessage {
	return repositories.BindStagingSecurityGroupMessage{
		GUID: guid,
		Spaces: slices.Collect(it.Map(slices.Values(b.Data), func(v RelationshipData) string {
			return v.GUID
		})),
	}
}
