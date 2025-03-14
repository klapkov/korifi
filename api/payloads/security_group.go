package payloads

import (
	"fmt"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"

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
	DisplayName     string                                `json:"name"`
	Rules           []korifiv1alpha1.SecurityGroupRule    `json:"rules"`
	GloballyEnabled korifiv1alpha1.SecurityGroupWorkloads `json:"globally_enabled"`
	Relationships   SecurityGroupRelationships            `json:"relationships"`
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

// validateSecurityGroupRules validates a slice of SecurityGroupRule
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
			return fmt.Errorf("Rules[%d]: ports are required for protocols of type TCP and UDP, ports must be a valid single port, comma separated list of ports, or range of ports, formatted as a string", i)
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

// validateRuleDestination validates that the destination is a valid IPv4 address, CIDR, or IP range
func validateRuleDestination(destination string) error {
	// Check for single IPv4 address
	if ip := net.ParseIP(destination); ip != nil && ip.To4() != nil {
		return nil
	}

	// Check for CIDR notation
	if ip, ipnet, err := net.ParseCIDR(destination); err == nil && ip.To4() != nil {
		ones, _ := ipnet.Mask.Size()
		if ones >= 1 && ones <= 32 {
			return nil
		}
	}

	// Check for IP range
	parts := strings.Split(destination, "-")
	if len(parts) == 2 {
		ip1 := net.ParseIP(parts[0])
		ip2 := net.ParseIP(parts[1])
		if ip1 != nil && ip1.To4() != nil && ip2 != nil && ip2.To4() != nil {
			return nil
		}
	}

	return fmt.Errorf("The Destination: %s is not in a valid format", destination)
}

// isValidPort checks if a string represents a valid port number (1-65535, no leading zeros)
func isValidPort(portStr string) bool {
	if len(portStr) == 0 || portStr[0] == '0' {
		return false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return false
	}
	return port >= 1 && port <= 65535
}

// validateRulePorts validates that the ports string is a valid single port, comma-separated list, or range
func validateRulePorts(ports string) error {
	if len(ports) == 0 {
		return nil
	}

	if strings.Count(ports, "-") == 1 && !strings.Contains(ports, ",") {
		// Port range (e.g., "80-90")
		parts := strings.Split(ports, "-")
		if len(parts) == 2 && isValidPort(parts[0]) && isValidPort(parts[1]) {
			return nil
		}
	} else {
		// Single port or comma-separated list (e.g., "80" or "80,443,8080")
		parts := strings.Split(ports, ",")
		for _, part := range parts {
			if !isValidPort(part) {
				return fmt.Errorf("invalid port: %s", part)
			}
		}
		return nil
	}

	return fmt.Errorf("The ports: %s is not in a valid format", ports)
}
