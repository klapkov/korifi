package payloads

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"

	"code.cloudfoundry.org/korifi/api/payloads/parse"
	"code.cloudfoundry.org/korifi/api/repositories"
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"github.com/BooleanCat/go-functional/v2/it"
	jellidation "github.com/jellydator/validation"
)

type SecurityGroupRelationships struct {
	RunningSpaces ToManyRelationship `json:"running_spaces"`
	StagingSpaces ToManyRelationship `json:"staging_spaces"`
}

type SecurityGroupCreate struct {
	DisplayName     string                             `json:"name"`
	Rules           []korifiv1alpha1.SecurityGroupRule `json:"rules"`
	GloballyEnabled korifiv1alpha1.GloballyEnabled     `json:"globally_enabled"`
	Relationships   SecurityGroupRelationships         `json:"relationships"`
}

func (c SecurityGroupCreate) Validate() error {
	return jellidation.ValidateStruct(&c,
		jellidation.Field(&c.DisplayName, jellidation.Required),
		jellidation.Field(&c.Rules, jellidation.Required, jellidation.By(validateSecurityGroupRules)),
	)
}

func (c SecurityGroupCreate) ToMessage() repositories.CreateSecurityGroupMessage {
	spaces := make(map[string]korifiv1alpha1.SecurityGroupWorkloads)

	for _, guid := range c.Relationships.RunningSpaces.CollectGUIDs() {
		workloads := spaces[guid]
		workloads.Running = true
		spaces[guid] = workloads
	}

	for _, guid := range c.Relationships.StagingSpaces.CollectGUIDs() {
		workloads := spaces[guid]
		workloads.Staging = true
		spaces[guid] = workloads
	}

	return repositories.CreateSecurityGroupMessage{
		DisplayName:     c.DisplayName,
		Rules:           c.Rules,
		GloballyEnabled: c.GloballyEnabled,
		Spaces:          spaces,
	}
}

type SecurityGroupList struct {
	GUIDs                  string `json:"guids"`
	Names                  string `json:"names"`
	GloballyEnabledStaging *bool  `json:"globally_enabled_staging"`
	GloballyEnabledRunning *bool  `json:"globally_enabled_running"`
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
	DisplayName     string                               `json:"name"`
	GloballyEnabled korifiv1alpha1.GloballyEnabledUpdate `json:"globally_enabled"`
	Rules           []korifiv1alpha1.SecurityGroupRule   `json:"rules"`
}

func (u SecurityGroupUpdate) Validate() error {
	return jellidation.ValidateStruct(&u,
		jellidation.Field(&u.Rules, jellidation.By(validateSecurityGroupRules)),
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

func validateSecurityGroupRules(value any) error {
	rules := value.([]korifiv1alpha1.SecurityGroupRule)

	for i, rule := range rules {
		if len(rule.Protocol) != 0 {
			if rule.Protocol != korifiv1alpha1.ProtocolALL && rule.Protocol != korifiv1alpha1.ProtocolTCP && rule.Protocol != korifiv1alpha1.ProtocolUDP {
				return fmt.Errorf("Rules[%d]: protocol %s not supported", i, rule.Protocol)
			}
		}

		if rule.Protocol == korifiv1alpha1.ProtocolALL && len(rule.Ports) != 0 {
			return fmt.Errorf("Rules[%d]: ports are not allowed for protocols of type all", i)
		}

		if (rule.Protocol == korifiv1alpha1.ProtocolTCP || rule.Protocol == korifiv1alpha1.ProtocolUDP) && len(rule.Ports) == 0 {
			return fmt.Errorf("Rules[%d]: ports are required for protocols of type TCP and UDP, ports must be a valid single port, comma separated list of ports, or range or ports, formatted as a string", i)
		}

		if err := validateRuleDestination(rule.Destination); err != nil {
			return fmt.Errorf("Rules[%d]: %w", i, err)
		}

		if err := validateRulePorts(rule.Ports); err != nil {
			return fmt.Errorf("Rules[%d]: %w", i, err)
		}
	}

	return nil
}

func validateRuleDestination(destination string) error {
	destIPRegex := regexp.MustCompile(
		`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.){3}(25[0-5]|(2[0-4]|1\d|[1-9]|)\d)$`,
	)
	cidrRegex := regexp.MustCompile(
		`^((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.){2}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\/([1-9]|[12][0-9]|3[0-2]))$`,
	)
	rangeRegex := regexp.MustCompile(
		`^(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.){2}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])-(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.((25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])\.){2}(25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$`,
	)

	if destIPRegex.MatchString(destination) || cidrRegex.MatchString(destination) || rangeRegex.MatchString(destination) {
		return nil
	}
	return fmt.Errorf("The Destination: %s is not in a valid format", destination)
}

func validateRulePorts(ports string) error {
	if len(ports) == 0 {
		return nil
	}

	singlePortRegex := regexp.MustCompile(
		`^([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$`,
	)

	multiplePortRegex := regexp.MustCompile(
		`^([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])(,([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5]))*$`,
	)

	rangeRegex := regexp.MustCompile(
		`^([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])-([1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])$`,
	)

	if singlePortRegex.MatchString(ports) || multiplePortRegex.MatchString(ports) || rangeRegex.MatchString(ports) {
		return nil
	}

	return fmt.Errorf("The ports: %s is not in a valid format", ports)
}
