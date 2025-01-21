package repositories

import (
	"context"
	"fmt"
	"slices"
	"time"

	"code.cloudfoundry.org/korifi/api/authorization"
	apierrors "code.cloudfoundry.org/korifi/api/errors"
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/tools"
	"code.cloudfoundry.org/korifi/tools/k8s"
	"github.com/BooleanCat/go-functional/v2/it"
	"github.com/BooleanCat/go-functional/v2/it/itx"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const SecurityGroupResourceType = "Security Group"

type SecurityGroupRepo struct {
	userClientFactory    authorization.UserClientFactory
	rootNamespace        string
	namespacePermissions *authorization.NamespacePermissions
	namespaceRetriever   NamespaceRetriever
}

func NewSecurityGroupRepo(
	userClientFactory authorization.UserClientFactory,
	rootNamespace string,
	namespacePermissions *authorization.NamespacePermissions,
	namespaceRetriever NamespaceRetriever,
) *SecurityGroupRepo {
	return &SecurityGroupRepo{
		userClientFactory:    userClientFactory,
		rootNamespace:        rootNamespace,
		namespacePermissions: namespacePermissions,
		namespaceRetriever:   namespaceRetriever,
	}
}

type CreateSecurityGroupMessage struct {
	DisplayName     string
	Rules           []korifiv1alpha1.SecurityGroupRule
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
		tools.EmptyOrContainsAllOf(m.RunningSpaceGUIDs, cfSecurityGroup.Spec.RunningSpaces) &&
		tools.EmptyOrContainsAllOf(m.StagingSpaceGUIDs, cfSecurityGroup.Spec.StagingSpaces)
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
	cfSecurityGroup.Spec.RunningSpaces = append(cfSecurityGroup.Spec.RunningSpaces, m.Spaces...)

}

type BindStagingSecurityGroupMessage struct {
	GUID   string
	Spaces []string
}

func (m *BindStagingSecurityGroupMessage) apply(cfSecurityGroup *korifiv1alpha1.CFSecurityGroup) {
	cfSecurityGroup.Spec.StagingSpaces = append(cfSecurityGroup.Spec.StagingSpaces, m.Spaces...)

}

type UnbindRunningSecurityGroupMessage struct {
	GUID      string
	SpaceGUID string
}

func (m *UnbindRunningSecurityGroupMessage) apply(cfSecurityGroup *korifiv1alpha1.CFSecurityGroup) {
	for i, space := range cfSecurityGroup.Spec.RunningSpaces {
		if space == m.SpaceGUID {
			cfSecurityGroup.Spec.RunningSpaces = append(cfSecurityGroup.Spec.RunningSpaces[:i], cfSecurityGroup.Spec.RunningSpaces[i+1:]...)
		}
	}
}

type UnbindStagingSecurityGroupMessage struct {
	GUID      string
	SpaceGUID string
}

func (m *UnbindStagingSecurityGroupMessage) apply(cfSecurityGroup *korifiv1alpha1.CFSecurityGroup) {
	for i, space := range cfSecurityGroup.Spec.StagingSpaces {
		if space == m.SpaceGUID {
			cfSecurityGroup.Spec.StagingSpaces = append(cfSecurityGroup.Spec.StagingSpaces[:i], cfSecurityGroup.Spec.StagingSpaces[i+1:]...)
		}
	}
}

type SecurityGroupRecord struct {
	GUID          string
	Name          string
	Rules         []korifiv1alpha1.SecurityGroupRule
	CreatedAt     time.Time
	DeletedAt     *time.Time
	RunningSpaces []string
	StagingSpaces []string
}

func (r *SecurityGroupRepo) GetSecurityGroup(ctx context.Context, authInfo authorization.Info, GUID string) (SecurityGroupRecord, error) {
	userClient, err := r.userClientFactory.BuildClient(authInfo)
	if err != nil {
		return SecurityGroupRecord{}, fmt.Errorf("failed to build user client: %w", err)
	}

	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      GUID,
		},
	}

	if err = userClient.Get(ctx, client.ObjectKeyFromObject(cfSecurityGroup), cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return toSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) CreateSecurityGroup(ctx context.Context, authInfo authorization.Info, message CreateSecurityGroupMessage) (SecurityGroupRecord, error) {
	userClient, err := r.userClientFactory.BuildClient(authInfo)
	if err != nil {
		return SecurityGroupRecord{}, fmt.Errorf("failed to build user client: %w", err)
	}

	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      uuid.NewString(),
		},
		Spec: korifiv1alpha1.CFSecurityGroupSpec{
			DisplayName:     message.DisplayName,
			Rules:           message.Rules,
			GloballyEnabled: message.GloballyEnabled,
		},
	}

	if err = userClient.Create(ctx, cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return toSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) ListSecurityGroups(ctx context.Context, authInfo authorization.Info, message ListSecurityGroupMessage) ([]SecurityGroupRecord, error) {
	userClient, err := r.userClientFactory.BuildClient(authInfo)
	if err != nil {
		return []SecurityGroupRecord{}, fmt.Errorf("failed to build user client: %w", err)
	}

	securityGroupList := &korifiv1alpha1.CFSecurityGroupList{}
	if err = userClient.List(ctx, securityGroupList, client.InNamespace(r.rootNamespace)); err != nil {
		return []SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	filteredSecurityGroups := itx.FromSlice(securityGroupList.Items).Filter(message.matches)
	return slices.Collect(it.Map(filteredSecurityGroups, toSecurityGroupRecord)), nil
}

func (r *SecurityGroupRepo) UpdateSecurityGroup(ctx context.Context, authInfo authorization.Info, message UpdateSecurityGroupMessage) (SecurityGroupRecord, error) {
	userClient, err := r.userClientFactory.BuildClient(authInfo)
	if err != nil {
		return SecurityGroupRecord{}, fmt.Errorf("failed to build user client: %w", err)
	}

	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err = userClient.Get(ctx, client.ObjectKeyFromObject(cfSecurityGroup), cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err = k8s.PatchResource(ctx, userClient, cfSecurityGroup, func() {
		message.apply(cfSecurityGroup)
	}); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return toSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) BindRunningSecurityGroup(ctx context.Context, authInfo authorization.Info, message BindRunningSecurityGroupMessage) (SecurityGroupRecord, error) {
	userClient, err := r.userClientFactory.BuildClient(authInfo)
	if err != nil {
		return SecurityGroupRecord{}, fmt.Errorf("failed to build user client: %w", err)
	}

	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err = userClient.Get(ctx, client.ObjectKeyFromObject(cfSecurityGroup), cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err = k8s.PatchResource(ctx, userClient, cfSecurityGroup, func() {
		message.apply(cfSecurityGroup)
	}); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return toSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) BindStagingSecurityGroup(ctx context.Context, authInfo authorization.Info, message BindStagingSecurityGroupMessage) (SecurityGroupRecord, error) {
	userClient, err := r.userClientFactory.BuildClient(authInfo)
	if err != nil {
		return SecurityGroupRecord{}, fmt.Errorf("failed to build user client: %w", err)
	}

	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err = userClient.Get(ctx, client.ObjectKeyFromObject(cfSecurityGroup), cfSecurityGroup); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err = k8s.PatchResource(ctx, userClient, cfSecurityGroup, func() {
		message.apply(cfSecurityGroup)
	}); err != nil {
		return SecurityGroupRecord{}, apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return toSecurityGroupRecord(*cfSecurityGroup), nil
}

func (r *SecurityGroupRepo) UnbindRunningSecurityGroup(ctx context.Context, authInfo authorization.Info, message UnbindRunningSecurityGroupMessage) error {
	userClient, err := r.userClientFactory.BuildClient(authInfo)
	if err != nil {
		return fmt.Errorf("failed to build user client: %w", err)
	}

	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err = userClient.Get(ctx, client.ObjectKeyFromObject(cfSecurityGroup), cfSecurityGroup); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err = k8s.PatchResource(ctx, userClient, cfSecurityGroup, func() {
		message.apply(cfSecurityGroup)
	}); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return nil
}

func (r *SecurityGroupRepo) UnbindStagingSecurityGroup(ctx context.Context, authInfo authorization.Info, message UnbindStagingSecurityGroupMessage) error {
	userClient, err := r.userClientFactory.BuildClient(authInfo)
	if err != nil {
		return fmt.Errorf("failed to build user client: %w", err)
	}

	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      message.GUID,
		},
	}

	if err = userClient.Get(ctx, client.ObjectKeyFromObject(cfSecurityGroup), cfSecurityGroup); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	if err = k8s.PatchResource(ctx, userClient, cfSecurityGroup, func() {
		message.apply(cfSecurityGroup)
	}); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return nil
}

func (r *SecurityGroupRepo) DeleteSecurityGroup(ctx context.Context, authInfo authorization.Info, GUID string) error {
	userClient, err := r.userClientFactory.BuildClient(authInfo)
	if err != nil {
		return fmt.Errorf("failed to build user client: %w", err)
	}

	cfSecurityGroup := &korifiv1alpha1.CFSecurityGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.rootNamespace,
			Name:      GUID,
		},
	}

	if err := userClient.Delete(ctx, cfSecurityGroup); err != nil {
		return apierrors.FromK8sError(err, SecurityGroupResourceType)
	}

	return nil
}

func toSecurityGroupRecord(cfSecurityGroup korifiv1alpha1.CFSecurityGroup) SecurityGroupRecord {
	return SecurityGroupRecord{
		GUID:      cfSecurityGroup.Name,
		Name:      cfSecurityGroup.Spec.DisplayName,
		Rules:     cfSecurityGroup.Spec.Rules,
		CreatedAt: cfSecurityGroup.CreationTimestamp.Time,
		// UpdatedAt:     getLastUpdatedTime(&),
		DeletedAt:     golangTime(cfSecurityGroup.DeletionTimestamp),
		RunningSpaces: cfSecurityGroup.Spec.RunningSpaces,
		StagingSpaces: cfSecurityGroup.Spec.StagingSpaces,
	}
}
