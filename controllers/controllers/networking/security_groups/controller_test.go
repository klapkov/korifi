package securitygroups_test

import (
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CFSecurityGroupReconciler Integration Tests", func() {
	var (
		cfSecurityGroup *korifiv1alpha1.CFSecurityGroup
		networkPolicy   *v1.NetworkPolicy
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
				Spaces: map[string]korifiv1alpha1.SecurityGroupWorkloads{
					testNamespace: {Running: true, Staging: false},
				},
			},
		}

		Expect(adminClient.Create(ctx, cfSecurityGroup)).To(Succeed())

	})

	It("sets the observed generation in the cfapp status", func() {
		Eventually(func(g Gomega) {
			g.Expect(adminClient.Get(ctx, client.ObjectKeyFromObject(cfSecurityGroup), cfSecurityGroup)).To(Succeed())
			g.Expect(cfSecurityGroup.Status.ObservedGeneration).To(BeEquivalentTo(cfSecurityGroup.Generation))
		}).Should(Succeed())
	})

	It("creates a network policy", func() {
		Eventually(func(g Gomega) {
			g.Expect(adminClient.Get(ctx, types.NamespacedName{Name: cfSecurityGroup.Name, Namespace: testNamespace}, networkPolicy)).To(Succeed())

		}).Should(Succeed())
	})
})
