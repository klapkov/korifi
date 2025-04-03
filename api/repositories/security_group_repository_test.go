package repositories_test

import (
	apierrors "code.cloudfoundry.org/korifi/api/errors"
	"code.cloudfoundry.org/korifi/api/repositories"
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/tests/matchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("SecurityGroupRepo", func() {
	var (
		repo  *repositories.SecurityGroupRepo
		org   *korifiv1alpha1.CFOrg
		space *korifiv1alpha1.CFSpace
	)

	BeforeEach(func() {
		repo = repositories.NewSecurityGroupRepo(userClientFactory, rootNamespace)
		org = createOrgWithCleanup(ctx, prefixedGUID("org"))
		space = createSpaceWithCleanup(ctx, org.Name, prefixedGUID("space"))
	})

	Describe("CreateSecurityGroup", func() {
		var (
			securityGroupRecord        repositories.SecurityGroupRecord
			securityGroupCreateMessage repositories.CreateSecurityGroupMessage
			createErr                  error
		)

		BeforeEach(func() {
			securityGroupCreateMessage = repositories.CreateSecurityGroupMessage{
				DisplayName: "test-security-group",
				Rules: []korifiv1alpha1.SecurityGroupRule{
					{
						Protocol:    korifiv1alpha1.ProtocolTCP,
						Ports:       "80",
						Destination: "192.168.1.1",
					},
				},
				Spaces: map[string]korifiv1alpha1.SecurityGroupWorkloads{
					space.Name: {Running: true, Staging: true},
				},
			}
		})

		JustBeforeEach(func() {
			securityGroupRecord, createErr = repo.CreateSecurityGroup(ctx, authInfo, securityGroupCreateMessage)
		})

		It("errors with forbidden for users with no permissions", func() {
			Expect(createErr).To(matchers.WrapErrorAssignableToTypeOf(apierrors.ForbiddenError{}))
		})

		When("the user is a CF admin", func() {
			BeforeEach(func() {
				createRoleBinding(ctx, userName, adminRole.Name, rootNamespace)
			})

			It("creates a CFSecurityGroup successfully", func() {
				Expect(createErr).ToNot(HaveOccurred())

				Expect(securityGroupRecord.GUID).To(matchers.BeValidUUID())
				Expect(securityGroupRecord.Name).To(Equal("test-security-group"))
				Expect(securityGroupRecord.GloballyEnabled.Running).To(BeFalse())
				Expect(securityGroupRecord.GloballyEnabled.Staging).To(BeFalse())
				Expect(securityGroupRecord.RunningSpaces).To(ConsistOf(space.Name))
				Expect(securityGroupRecord.StagingSpaces).To(ConsistOf(space.Name))
				Expect(securityGroupRecord.Rules).To(ConsistOf(korifiv1alpha1.SecurityGroupRule{
					Protocol:    korifiv1alpha1.ProtocolTCP,
					Ports:       "80",
					Destination: "192.168.1.1",
				}))
			})
		})
	})

	Describe("BindRunningSecurityGroup", func() {
		var (
			cfSecurityGroup     *korifiv1alpha1.CFSecurityGroup
			securityGroupRecord repositories.SecurityGroupRecord
			bindMessage         repositories.BindSecurityGroupMessage
			bindErr             error
		)

		BeforeEach(func() {
			cfSecurityGroup = &korifiv1alpha1.CFSecurityGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      prefixedGUID("sg"),
					Namespace: rootNamespace,
				},
				Spec: korifiv1alpha1.CFSecurityGroupSpec{
					DisplayName: "test-security-group",
					Rules: []korifiv1alpha1.SecurityGroupRule{
						{
							Protocol:    korifiv1alpha1.ProtocolTCP,
							Ports:       "80",
							Destination: "192.168.1.1",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cfSecurityGroup)).To(Succeed())

			bindMessage = repositories.BindSecurityGroupMessage{
				GUID:   cfSecurityGroup.Name,
				Spaces: []string{space.Name},
			}
		})

		JustBeforeEach(func() {
			securityGroupRecord, bindErr = repo.BindRunningSecurityGroup(ctx, authInfo, bindMessage)
		})

		It("errors with forbidden for users with no permissions", func() {
			Expect(bindErr).To(matchers.WrapErrorAssignableToTypeOf(apierrors.ForbiddenError{}))
		})

		When("the user is a CF admin", func() {
			BeforeEach(func() {
				createRoleBinding(ctx, userName, adminRole.Name, rootNamespace)
			})

			It("binds the running space to the security group", func() {
				Expect(bindErr).ToNot(HaveOccurred())
				Expect(securityGroupRecord.RunningSpaces).To(ConsistOf(space.Name))
			})
		})
	})

	Describe("BindStagingSecurityGroup", func() {
		var (
			cfSecurityGroup     *korifiv1alpha1.CFSecurityGroup
			securityGroupRecord repositories.SecurityGroupRecord
			bindMessage         repositories.BindSecurityGroupMessage
			bindErr             error
		)

		BeforeEach(func() {
			cfSecurityGroup = &korifiv1alpha1.CFSecurityGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      prefixedGUID("sg"),
					Namespace: rootNamespace,
				},
				Spec: korifiv1alpha1.CFSecurityGroupSpec{
					DisplayName: "test-security-group",
					Rules: []korifiv1alpha1.SecurityGroupRule{
						{
							Protocol:    korifiv1alpha1.ProtocolTCP,
							Ports:       "80",
							Destination: "192.168.1.1",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cfSecurityGroup)).To(Succeed())

			bindMessage = repositories.BindSecurityGroupMessage{
				GUID:   cfSecurityGroup.Name,
				Spaces: []string{space.Name},
			}
		})

		JustBeforeEach(func() {
			securityGroupRecord, bindErr = repo.BindStagingSecurityGroup(ctx, authInfo, bindMessage)
		})

		It("errors with forbidden for users with no permissions", func() {
			Expect(bindErr).To(matchers.WrapErrorAssignableToTypeOf(apierrors.ForbiddenError{}))
		})

		When("the user is a CF admin", func() {
			BeforeEach(func() {
				createRoleBinding(ctx, userName, adminRole.Name, rootNamespace)
			})

			It("binds the staging space to the security group", func() {
				Expect(bindErr).ToNot(HaveOccurred())
				Expect(securityGroupRecord.StagingSpaces).To(ConsistOf(space.Name))
			})
		})
	})
})
