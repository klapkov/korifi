package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CFWorkloadTypeApp              = "app"
	CFWorkloadTypeBuild            = "build"
	CFSecurityGroupTypeLabel       = "korifi.cloudfoundry.org/security-group-type"
	CFSecurityGroupNameLabel       = "korifi.cloudfoundry.org/security-group-name"
	CFSecurityGroupTypeGlobal      = "global"
	CFSecurityGroupTypeSpaceScoped = "space-scoped"
	CFSecurityGroupFinalizerName   = "cfSecurityGroup.korifi.cloudfoundry.org"
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

type GloballyEnabled struct {
	Running bool `json:"running,omitempty"`
	Staging bool `json:"staging,omitempty"`
}

type GloballyEnabledUpdate struct {
	Running *bool `json:"running,omitempty"`
	Staging *bool `json:"staging,omitempty"`
}

type CFSecurityGroupSpec struct {
	DisplayName string              `json:"displayName"`
	Rules       []SecurityGroupRule `json:"rules"`
	//+kubebuilder:validation:Optional
	RunningSpaces []string `json:"running_spaces"`
	//+kubebuilder:validation:Optional
	StagingSpaces   []string        `json:"staging_spaces"`
	GloballyEnabled GloballyEnabled `json:"globally_enabled"`
}

type CFSecurityGroupStatus struct {
	//+kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:subresource:status
//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="DisplayName",type=string,JSONPath=`.spec.displayName`
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CFSecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CFSecurityGroupSpec `json:"spec,omitempty"`

	Status CFSecurityGroupStatus `json:"status,omitempty"`
}

func (g *CFSecurityGroup) StatusConditions() *[]metav1.Condition {
	return &g.Status.Conditions
}

//+kubebuilder:object:root=true
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CFSecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CFSecurityGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CFSecurityGroup{}, &CFSecurityGroupList{})
}
