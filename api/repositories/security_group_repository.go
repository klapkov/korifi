package repositories

import (
	"context"
	"slices"
	"time"

	"code.cloudfoundry.org/korifi/api/authorization"
	apierrors "code.cloudfoundry.org/korifi/api/errors"
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/tools"
	"github.com/BooleanCat/go-functional/v2/it"
	"github.com/BooleanCat/go-functional/v2/it/itx"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const SecurityGroupResourceType = "Security Group"

type SecurityGroupRepo struct {
	klient               Klient
	rootNamespace        string
	namespacePermissions *authorization.NamespacePermissions
	namespaceRetriever   NamespaceRetriever
}

func NewSecurityGroupRepo(
	klient Klient,
	rootNamespace string,
	namespacePermissions *authorization.NamespacePermissions,
	namespaceRetriever NamespaceRetriever,
) *SecurityGroupRepo {
	return &SecurityGroupRepo{
		klient:               klient,
		rootNamespace:        rootNamespace,
		namespacePermissions: namespacePermissions,
		namespaceRetriever:   namespaceRetriever,
	}
}

type CreateSecurityGroupMessage struct {
	DisplayName     string
	Rules           []korifiv1alpha1.SecurityGroupRule
	Spaces          map[string]korifiv1alpha1.SecurityGroupWorkloads
	GloballyEnabled korifiv1alpha1.GloballyEnabled
}

type ListSecurityGroupMessage struct {
	GUIDs                  []string
	Names                  []string
	GloballyEnabledStaging *bool
	GloballyEnabledRunning *bool
	RunningSpaceGUIDs      []string
	StagingSpaceGUIDs      []string
}

func (m *ListSecurityGroupMessage) matches(cfSecurityGroup korifiv1alpha1.CFSecurityGroup) bool {
	return tools.EmptyOrContains(m.GUIDs, cfSecurityGroup.Name) &&
		tools.EmptyOrContains(m.Names, cfSecurityGroup.Spec.DisplayName) &&
		tools.NilOrEquals(m.GloballyEnabledStaging, cfSecurityGroup.Spec.GloballyEnabled.Staging) &&
		tools.NilOrEquals(m.GloballyEnabledRunning, cfSecurityGroup.Spec.GloballyEnabled.Running) &&
		tools.EmptyOrContainsAll(m.RunningSpaceGUIDs, cfSecurityGroup.Spec.Spaces, func(s korifiv1alpha1.SecurityGroupWorkloads) bool { return s.Running }) &&
		tools.EmptyOrContainsAll(m.StagingSpaceGUIDs, cfSecurityGroup.Spec.Spaces, func(s korifiv1alpha1.SecurityGroupWorkloads) bool { return s.Staging })
}

type UpdateSecurityGroupMessage struct {
	GUID            string
	DisplayName     string
	Rules           []korifiv1alpha1.SecurityGroupRule
	GloballyEnabled korifiv1alpha1.GloballyEnabledUpdate
}

func (m *UpdateSecurityGroupMessage) apply(cfSecurityGroup *korifiv1alpha1.CFSecurityGroup) {
	if m.DisplayName != "" {
		cfSecurityGroup.Spec.DisplayName = m.DisplayName
	}

	if m.GloballyEnabled.Running != nil {
		cfSecurityGroup.Spec.GloballyEnabled.Running = *m.GloballyEnabled.Running
	}

	if m.GloballyEnabled.Staging != nil {
		cfSecurityGroup.Spec.GloballyEnabled.Staging = *m.GloballyEnabled.Staging
	}

	if len(m.Rules) > 0 {
		cfSecurityGroup.Spec.Rules = m.Rules
	}
}

type BindRunningSecurityGroupMessage struct {
	GUID   string
	Spaces []string
}

func (m *BindRunningSecurityGroupMessage) apply(cfSecurityGroup *korifiv1alpha1.CFSecurityGroup) {
	if cfSecurityGroup.Spec.Spaces == nil {
		cfSecurityGroup.Spec.Spaces = make(map[string]korifiv1alpha1.SecurityGroupWorkloads)
	}

	for _, space := range m.Spaces {
		workloads := cfSecurityGroup.Spec.Spaces[space]
		workloads.Running = true
		cfSecurityGroup.Spec.Spaces[space] = workloads
	}
}

type BindStagingSecurityGroupMessage struct {
	GUID   string
	Spaces []string
}

func (m *BindStagingSecurityGroupMessage) apply(cfSecurityGroup *korifiv1alpha1.CFSecurityGroup) {
	if cfSecurityGroup.Spec.Spaces == nil {
		cfSecurityGroup.Spec.Spaces = make(map[string]korifiv1alpha1.SecurityGroupWorkloads)
	}

	for _, space := range m.Spaces {
		workloads := cfSecurityGroup.Spec.Spaces[space]
		workloads.Staging = true
		cfSecurityGroup.Spec.Spaces[space] = workloads
	}
}

type UnbindRunningSecurityGroupMessage struct {
	GUID      string
	SpaceGUID string
}

func (m *UnbindRunningSecurityGroupMessage) apply(cfSecurityGroup *korifiv1alpha1.CFSecurityGroup) {
	if space, exists := cfSecurityGroup.Spec.Spaces[m.SpaceGUID]; exists {
		space.Running = false

		if !space.Running && !space.Staging {
			delete(cfSecurityGroup.Spec.Spaces, m.SpaceGUID)
		} else {
			cfSecurityGroup.Spec.Spaces[m.SpaceGUID] = space
		}
	}
}

type UnbindStagingSecurityGroupMessage struct {
	GUID      string
	SpaceGUID string
}

func (m *UnbindStagingSecurityGroupMessage) apply(cfSecurityGroup *korifiv1alpha1.CFSecurityGroup) {
	if space, exists := cfSecurityGroup.Spec.Spaces[m.SpaceGUID]; exists {
		space.Staging = false
		if !space.Running && !space.Staging {
			delete(cfSecurityGroup.Spec.Spaces, m.SpaceGUID)
		} else {
			cfSecurityGroup.Spec.Spaces[m.SpaceGUID] = space
		}
	}
}

type SecurityGroupRecord struct {
	GUID            string
	CreatedAt       time.Time
	UpdatedAt       *time.Time
	DeletedAt       *time.Time
	Name            string
	Rules           []korifiv1alpha1.SecurityGroupRule
	GloballyEnabled korifiv1alpha1.GloballyEnabled
	RunningSpaces   []string
	StagingSpaces   []string
}

func (r *SecurityGroupRepo) GetSecurityGroup(ctx context.Context, authInfo authorization.Info, GUID string) (SecurityGroupRecord, error) {
	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      GUID,
		},
	}

	if err := r.klient.Get(ctx, cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return ToSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) CreateSecurityGroup(ctx context.Context, authInfo authorization.Info, message CreateSecurityGroupMessage) (SecurityGroupRecord, error) {
	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      uuid.NewString(),
		},
		Spec: korifiv1alpha1.CFSecurityGroupSpec{
			DisplayName:     message.DisplayName,
			Rules:           message.Rules,
			Spaces:          message.Spaces,
			GloballyEnabled: message.GloballyEnabled,
		},
	}

	if err := r.klient.Create(ctx, cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return ToSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) ListSecurityGroups(ctx context.Context, authInfo authorization.Info, message ListSecurityGroupMessage) ([]SecurityGroupRecord, error) {
	securityGroupList := &korifiv1alpha1.CFSecurityGroupList{}
	if err := r.klient.List(ctx, securityGroupList, InNamespace(r.rootNamespace)); err != nil {
		return []SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	filteredSecurityGroups := itx.FromSlice(securityGroupList.Items).Filter(message.matches)
	return slices.Collect(it.Map(filteredSecurityGroups, ToSecurityGroupRecord)), nil
}

func (r *SecurityGroupRepo) UpdateSecurityGroup(ctx context.Context, authInfo authorization.Info, message UpdateSecurityGroupMessage) (SecurityGroupRecord, error) {
	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err := r.klient.Get(ctx, cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err := r.klient.Patch(ctx, cfSecurityGroup, func() error {
		message.apply(cfSecurityGroup)
		return nil
	}); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return ToSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) BindRunningSecurityGroup(ctx context.Context, authInfo authorization.Info, message BindRunningSecurityGroupMessage) (SecurityGroupRecord, error) {
	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err := r.klient.Get(ctx, cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err := r.klient.Patch(ctx, cfSecurityGroup, func() error {
		message.apply(cfSecurityGroup)
		return nil
	}); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return ToSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) BindStagingSecurityGroup(ctx context.Context, authInfo authorization.Info, message BindStagingSecurityGroupMessage) (SecurityGroupRecord, error) {
	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err := r.klient.Get(ctx, cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err := r.klient.Patch(ctx, cfSecurityGroup, func() error {
		message.apply(cfSecurityGroup)
		return nil
	}); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return ToSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) UnbindRunningSecurityGroup(ctx context.Context, authInfo authorization.Info, message UnbindRunningSecurityGroupMessage) error {
	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err := r.klient.Get(ctx, cfSecurityGroup); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err := r.klient.Patch(ctx, cfSecurityGroup, func() error {
		message.apply(cfSecurityGroup)
		return nil
	}); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return nil
}

func (r *SecurityGroupRepo) UnbindStagingSecurityGroup(ctx context.Context, authInfo authorization.Info, message UnbindStagingSecurityGroupMessage) error {
	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err := r.klient.Get(ctx, cfSecurityGroup); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err := r.klient.Patch(ctx, cfSecurityGroup, func() error {
		message.apply(cfSecurityGroup)
		return nil
	}); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return nil
}

func (r *SecurityGroupRepo) DeleteSecurityGroup(ctx context.Context, authInfo authorization.Info, GUID string) error {
	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      GUID,
		},
	}

	if err := r.klient.Delete(ctx, cfSecurityGroup); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return nil
}

func ToSecurityGroupRecord(cfSecurityGroup korifiv1alpha1.CFSecurityGroup) SecurityGroupRecord {
	runningSpaces := []string{}
	stagingSpaces := []string{}

	for space, workloads := range cfSecurityGroup.Spec.Spaces {
		if workloads.Running {
			runningSpaces = append(runningSpaces, space)
		}
		if workloads.Staging {
			stagingSpaces = append(stagingSpaces, space)
		}
	}

	return SecurityGroupRecord{
		GUID:            cfSecurityGroup.Name,
		CreatedAt:       cfSecurityGroup.CreationTimestamp.Time,
		DeletedAt:       golangTime(cfSecurityGroup.DeletionTimestamp),
		Name:            cfSecurityGroup.Spec.DisplayName,
		GloballyEnabled: cfSecurityGroup.Spec.GloballyEnabled,
		Rules:           cfSecurityGroup.Spec.Rules,
		UpdatedAt:       getLastUpdatedTime(&cfSecurityGroup),
		RunningSpaces:   runningSpaces,
		StagingSpaces:   stagingSpaces,
	}
}
