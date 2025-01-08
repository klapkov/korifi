package payloads

import (
	"fmt"
	"net/url"
	"slices"

	"code.cloudfoundry.org/korifi/api/payloads/parse"
	"code.cloudfoundry.org/korifi/api/repositories"
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"github.com/BooleanCat/go-functional/v2/it"
	jellidation "github.com/jellydator/validation"
)

type SecurityGroupRule struct {
	Protocol    string `json:"protocol"`
	Destination string `json:"destination"`
	Ports       string `json:"ports,omitempty"`
	Type        int    `json:"type,omitempty"`
	Code        int    `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
	Log         bool   `json:"log,omitempty"`
}

type SecurityGroupWorkloads struct {
	Running bool `json:"running"`
	Staging bool `json:"staging"`
}

type SecurityGroupRelationships struct {
	RunningSpaces ToManyRelationship `json:"running_spaces"`
	StagingSpaces ToManyRelationship `json:"staging_spaces"`
}

type SecurityGroupCreate struct {
	DisplayName     string                     `json:"name"`
	Rules           []SecurityGroupRule        `json:"rules"`
	GloballyEnabled SecurityGroupWorkloads     `json:"globally_enabled"`
	Relationships   SecurityGroupRelationships `json:"relationships"`
}

func (c SecurityGroupCreate) Validate() error {
	return jellidation.ValidateStruct(&c,
		jellidation.Field(&c.DisplayName, jellidation.Required),
		jellidation.Field(&c.Rules, jellidation.Required),
	)
}

func (c SecurityGroupCreate) ToMessage() repositories.CreateSecurityGroupMessage {
	rules := slices.Collect(it.Map(slices.Values(c.Rules), func(r SecurityGroupRule) repositories.SecurityGroupRule {
		return repositories.SecurityGroupRule{
			Protocol:    r.Protocol,
			Destination: r.Destination,
			Ports:       r.Ports,
			Type:        r.Type,
			Code:        r.Code,
			Description: r.Description,
			Log:         r.Log,
		}
	}))

	spaces := make(map[string]repositories.SecurityGroupWorkloads)
	runningSpaces := slices.Collect(it.Map(slices.Values(c.Relationships.RunningSpaces.Data), func(d RelationshipData) string { return d.GUID }))
	stagingSpaces := slices.Collect(it.Map(slices.Values(c.Relationships.StagingSpaces.Data), func(d RelationshipData) string { return d.GUID }))

	for _, guid := range runningSpaces {
		workloads := spaces[guid]
		workloads.Running = true
		spaces[guid] = workloads
	}

	for _, guid := range stagingSpaces {
		workloads := spaces[guid]
		workloads.Staging = true
		spaces[guid] = workloads
	}

	return repositories.CreateSecurityGroupMessage{
		DisplayName: c.DisplayName,
		Rules:       rules,
		GloballyEnabled: repositories.SecurityGroupWorkloads{
			Running: c.GloballyEnabled.Running,
			Staging: c.GloballyEnabled.Staging,
		},
		Spaces: spaces,
	}
}

type SecurityGroupList struct {
	GUIDs                  string `json:"guids"`
	Names                  string `json:"names"`
	GloballyEnabledRunning *bool  `json:"globally_enabled_running"`
	GloballyEnabledStaging *bool  `json:"globally_enabled_staging"`
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
		"per_page",
		"page",
		"order_by",
		"created_ats",
		"updated_ats",
	}
}

func (l *SecurityGroupList) DecodeFromURLValues(values url.Values) error {
	var err error

	l.GUIDs = values.Get("guids")
	l.Names = values.Get("names")
	globallyEnabledStaging, err := parseBool(values.Get("globally_enabled_staging"))
	if err != nil {
		return fmt.Errorf("failed to parse 'globally_enabled_staging' query parameter: %w", err)
	}
	globallyEnabledRunning, err := parseBool(values.Get("globally_enabled_running"))
	if err != nil {
		return fmt.Errorf("failed to parse 'globally_enabled_running' query parameter: %w", err)
	}
	l.GloballyEnabledStaging = globallyEnabledStaging
	l.GloballyEnabledRunning = globallyEnabledRunning
	l.RunningSpaceGUIDs = values.Get("running_space_guids")
	l.StagingSpaceGUIDs = values.Get("staging_space_guids")

	return nil
}

func (l SecurityGroupList) ToMessage() repositories.ListSecurityGroupMessage {
	return repositories.ListSecurityGroupMessage{
		GUIDs:                  parse.ArrayParam(l.GUIDs),
		Names:                  parse.ArrayParam(l.Names),
		GloballyEnabledStaging: l.GloballyEnabledStaging,
		GloballyEnabledRunning: l.GloballyEnabledRunning,
		RunningSpaceGUIDs:      parse.ArrayParam(l.RunningSpaceGUIDs),
		StagingSpaceGUIDs:      parse.ArrayParam(l.StagingSpaceGUIDs),
	}
}

type SecurityGroupUpdate struct {
	DisplayName     string                                      `json:"name"`
	GloballyEnabled korifiv1alpha1.SecurityGroupWorkloadsUpdate `json:"globally_enabled"`
	Rules           []korifiv1alpha1.SecurityGroupRule          `json:"rules"`
}

func (u SecurityGroupUpdate) Validate() error {
	return jellidation.ValidateStruct(&u,
		jellidation.Field(&u.Rules, jellidation.Required),
	)
}

func (u SecurityGroupUpdate) ToMessage(guid string) repositories.UpdateSecurityGroupMessage {
	return repositories.UpdateSecurityGroupMessage{
		GUID:            guid,
		DisplayName:     u.DisplayName,
		GloballyEnabled: u.GloballyEnabled,
		Rules:           u.Rules,
	}
}

type SecurityGroupBind struct {
	Data []RelationshipData `json:"data"`
}

func (b SecurityGroupBind) Validate() error {
	return jellidation.ValidateStruct(&b,
		jellidation.Field(&b.Data, jellidation.Required),
	)
}

func (b SecurityGroupBind) ToMessage(workload, guid string) repositories.BindSecurityGroupMessage {
	return repositories.BindSecurityGroupMessage{
		GUID: guid,
		Spaces: slices.Collect(it.Map(slices.Values(b.Data), func(v RelationshipData) string {
			return v.GUID
		})),
		Workload: workload,
	}
}
