package securitygroups_test

import (
	"fmt"

	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/tools"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	// . "github.com/onsi/gomega/gstruct"
	v3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/api/pkg/lib/numorstring"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CFSecurityGroupReconciler Integration Tests", func() {
	var (
		cfSecurityGroup *korifiv1alpha1.CFSecurityGroup
	)

	BeforeEach(func() {
		cfSecurityGroup = &korifiv1alpha1.CFSecurityGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: rootNamespace,
			},
			Spec: korifiv1alpha1.CFSecurityGroupSpec{
				DisplayName: "test-security-group",
				Rules: []korifiv1alpha1.SecurityGroupRule{{
					Protocol:    korifiv1alpha1.ProtocolTCP,
					Ports:       "80",
					Destination: "192.168.1.1",
				}},
			},
		}
	})

	JustBeforeEach(func() {
		Expect(adminClient.Create(ctx, cfSecurityGroup)).To(Succeed())
	})

	It("sets the observed generation in the cfsecuritygroup status", func() {
		Eventually(func(g Gomega) {
			g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cfSecurityGroup), cfSecurityGroup)).To(Succeed())
			g.Expect(cfSecurityGroup.Status.ObservedGeneration).To(BeEquivalentTo(cfSecurityGroup.Generation))
		}).Should(Succeed())
	})

	It("does not create a network policy when there are no spaces appended", func() {
		Eventually(func(g Gomega) {
			policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
			g.Expect(policy).To(BeNil())
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue())

			globalPolicy, err := calicoFakeClient.ProjectcalicoV3().GlobalNetworkPolicies().Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
			g.Expect(globalPolicy).To(BeNil())
			g.Expect(apierrors.IsNotFound(err)).To(BeTrue())

		}).Should(Succeed())

	})

	When("appending a space to the group", func() {
		BeforeEach(func() {
			cfSecurityGroup.Spec.Spaces = map[string]korifiv1alpha1.SecurityGroupWorkloads{testNamespace: {Running: true, Staging: false}}
		})

		It("creates a network policy", func() {
			Eventually(func(g Gomega) {
				policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(policy.Spec.Egress[0].Action).To(Equal(v3.Action("Allow")))
				g.Expect(policy.Spec.Types).To(Equal([]v3.PolicyType{"Egress"}))

			}).Should(Succeed())
		})

		When("we append a staging space", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.Spaces = map[string]korifiv1alpha1.SecurityGroupWorkloads{testNamespace: {Running: false, Staging: true}}
			})

			It("creates the proper selector", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Selector).To(Equal("korifi.cloudfoundry.org/workload-type in { 'build' }"))
				}).Should(Succeed())
			})
		})

		When("the space is both staging and running", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.Spaces = map[string]korifiv1alpha1.SecurityGroupWorkloads{testNamespace: {Running: true, Staging: true}}
			})

			It("creates the proper selector", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Selector).To(Equal("korifi.cloudfoundry.org/workload-type in { 'app', 'build' }"))
				}).Should(Succeed())
			})
		})

		When("the destination is an IP range", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.Rules = []korifiv1alpha1.SecurityGroupRule{{
					Protocol:    korifiv1alpha1.ProtocolTCP,
					Ports:       "80",
					Destination: "192.168.1.1-192.168.1.255",
				}}
			})

			It("creates multiple nets to cover the desired cidr range", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Egress[0].Destination.Nets).To(ConsistOf(
						"192.168.1.1/32",
						"192.168.1.2/31",
						"192.168.1.4/30",
						"192.168.1.8/29",
						"192.168.1.16/28",
						"192.168.1.32/27",
						"192.168.1.64/26",
						"192.168.1.128/25",
					))
				}).Should(Succeed())
			})
		})

		When("the destination is a single ip", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.Rules = []korifiv1alpha1.SecurityGroupRule{{
					Protocol:    korifiv1alpha1.ProtocolTCP,
					Ports:       "80",
					Destination: "192.168.1.1",
				}}
			})

			It("creates only one net", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Egress[0].Destination.Nets).To(ConsistOf("192.168.1.1/32"))
				}).Should(Succeed())
			})
		})

		When("the destination is a cidr", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.Rules = []korifiv1alpha1.SecurityGroupRule{{
					Protocol:    korifiv1alpha1.ProtocolTCP,
					Ports:       "80",
					Destination: "192.168.1.1/16",
				}}
			})

			It("creates the proper cidr", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Egress[0].Destination.Nets).To(ConsistOf("192.168.1.1/16"))
				}).Should(Succeed())
			})
		})

		When("we pass a single port", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.Rules = []korifiv1alpha1.SecurityGroupRule{{
					Protocol:    korifiv1alpha1.ProtocolTCP,
					Ports:       "80",
					Destination: "192.168.1.1/16",
				}}
			})

			It("creates the rule with the proper port", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Egress[0].Destination.Ports).To(Equal([]numorstring.Port{{
						MinPort: 80,
						MaxPort: 80,
					}}))
				}).Should(Succeed())
			})
		})

		When("we pass a range of ports", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.Rules = []korifiv1alpha1.SecurityGroupRule{{
					Protocol:    korifiv1alpha1.ProtocolTCP,
					Ports:       "80-90",
					Destination: "192.168.1.1/16",
				}}
			})

			It("creates the rule with the proper port range", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Egress[0].Destination.Ports).To(Equal([]numorstring.Port{{
						MinPort: 80,
						MaxPort: 90,
					}}))
				}).Should(Succeed())
			})
		})

		When("we pass a list of ports", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.Rules = []korifiv1alpha1.SecurityGroupRule{{
					Protocol:    korifiv1alpha1.ProtocolTCP,
					Ports:       "80,90,100",
					Destination: "192.168.1.1/16",
				}}
			})

			It("creates the rule with the proper port range", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Egress[0].Destination.Ports).To(Equal([]numorstring.Port{
						{
							MinPort: 80,
							MaxPort: 80,
						},
						{
							MinPort: 90,
							MaxPort: 90,
						},
						{
							MinPort: 100,
							MaxPort: 100,
						},
					}))
				}).Should(Succeed())
			})
		})

		When("we pass UDP as a protocol", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.Rules = []korifiv1alpha1.SecurityGroupRule{{
					Protocol:    korifiv1alpha1.ProtocolUDP,
					Ports:       "80",
					Destination: "192.168.1.1/16",
				}}
			})

			It("creates the rule with UDP protocol", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().NetworkPolicies(testNamespace).Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Egress[0].Protocol).To(Equal(tools.PtrTo(numorstring.Protocol{Type: 1, StrVal: numorstring.ProtocolUDP})))
				}).Should(Succeed())
			})
		})
	})

	When("the security group is global", func() {
		BeforeEach(func() {
			cfSecurityGroup.Spec.GloballyEnabled.Running = true
		})

		It("creates a global network policy", func() {
			Eventually(func(g Gomega) {
				policy, err := calicoFakeClient.ProjectcalicoV3().GlobalNetworkPolicies().Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(policy.Spec.Egress[0].Action).To(Equal(v3.Action("Allow")))
				g.Expect(policy.Spec.Types).To(Equal([]v3.PolicyType{"Egress"}))
				g.Expect(policy.Spec.Selector).To(Equal("korifi.cloudfoundry.org/workload-type in { 'app' }"))
				g.Expect(policy.Spec.NamespaceSelector).To(Equal("has(cloudfoundry.org/space-guid)"))

			}).Should(Succeed())
		})

		When("the security group is global for both running and staging", func() {
			BeforeEach(func() {
				cfSecurityGroup.Spec.GloballyEnabled.Staging = true
			})

			It("creates the proper selector", func() {
				Eventually(func(g Gomega) {
					policy, err := calicoFakeClient.ProjectcalicoV3().GlobalNetworkPolicies().Get(ctx, fmt.Sprint("default.", cfSecurityGroup.Name), metav1.GetOptions{})
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(policy.Spec.Selector).To(Equal("korifi.cloudfoundry.org/workload-type in { 'app', 'build' }"))
				}).Should(Succeed())
			})
		})
	})
})
