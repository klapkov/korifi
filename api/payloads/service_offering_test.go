package payloads_test

import (
	"code.cloudfoundry.org/korifi/api/payloads"
	"code.cloudfoundry.org/korifi/api/payloads/params"
	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/tools"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

var _ = Describe("ServiceOfferingGet", func() {
	DescribeTable("valid query",
		func(query string, expectedServiceOfferingGet payloads.ServiceOfferingGet) {
			actualServiceOfferingGet, decodeErr := decodeQuery[payloads.ServiceOfferingGet](query)

			Expect(decodeErr).NotTo(HaveOccurred())
			Expect(*actualServiceOfferingGet).To(Equal(expectedServiceOfferingGet))
		},
		Entry("fields[service_broker]", "fields[service_broker]=guid,name", payloads.ServiceOfferingGet{
			IncludeResourceRules: []params.IncludeResourceRule{{
				RelationshipPath: []string{"service_broker"},
				Fields:           []string{"guid", "name"},
			}},
		}),
	)

	DescribeTable("invalid query",
		func(query string, errMatcher types.GomegaMatcher) {
			_, decodeErr := decodeQuery[payloads.ServiceOfferingGet](query)
			Expect(decodeErr).To(errMatcher)
		},
		Entry("invalid service broker field", "fields[service_broker]=foo", MatchError(ContainSubstring("value must be one of"))),
		Entry("invalid fields", "fields[space]=foo", MatchError(ContainSubstring("unsupported query parameter: fields[space]"))),
	)
})

var _ = Describe("ServiceOfferingPatch", func() {
	var (
		patchPayload         payloads.ServiceOfferingPatch
		serviceOfferingPatch *payloads.ServiceOfferingPatch
		validatorErr         error
	)

	BeforeEach(func() {
		serviceOfferingPatch = new(payloads.ServiceOfferingPatch)

		patchPayload = payloads.ServiceOfferingPatch{
			Metadata: payloads.MetadataPatch{
				Annotations: map[string]*string{"ann1": tools.PtrTo("val_ann1")},
				Labels:      map[string]*string{"lab1": tools.PtrTo("val_lab1")},
			},
		}
	})

	JustBeforeEach(func() {
		validatorErr = validator.DecodeAndValidateJSONPayload(createJSONRequest(patchPayload), serviceOfferingPatch)
	})

	It("succeeds", func() {
		Expect(validatorErr).NotTo(HaveOccurred())
		Expect(serviceOfferingPatch).To(PointTo(Equal(patchPayload)))
	})

	When("nothing is set", func() {
		BeforeEach(func() {
			patchPayload = payloads.ServiceOfferingPatch{}
		})

		It("succeeds", func() {
			Expect(validatorErr).NotTo(HaveOccurred())
			Expect(serviceOfferingPatch).To(PointTo(Equal(patchPayload)))
		})
	})

	When("metadata is invalid", func() {
		BeforeEach(func() {
			patchPayload.Metadata.Labels["foo.cloudfoundry.org/bar"] = tools.PtrTo("baz")
		})

		It("returns an appropriate error", func() {
			expectUnprocessableEntityError(validatorErr, "label/annotation key cannot use the cloudfoundry.org domain")
		})
	})

	It("converts the patch message correctly", func() {
		msg := serviceOfferingPatch.ToMessage("offering-guid")

		Expect(msg.GUID).To(Equal("offering-guid"))
		Expect(msg.Metadata.Annotations).To(MatchAllKeys(Keys{
			"ann1": PointTo(Equal("val_ann1")),
		}))
		Expect(msg.Metadata.Labels).To(MatchAllKeys(Keys{
			"lab1": PointTo(Equal("val_lab1")),
		}))
	})
})

var _ = Describe("ServiceOfferingList", func() {
	DescribeTable("valid query",
		func(query string, expectedServiceOfferingList payloads.ServiceOfferingList) {
			actualServiceOfferingList, decodeErr := decodeQuery[payloads.ServiceOfferingList](query)

			Expect(decodeErr).NotTo(HaveOccurred())
			Expect(*actualServiceOfferingList).To(Equal(expectedServiceOfferingList))
		},
		Entry("names", "names=b1,b2", payloads.ServiceOfferingList{Names: "b1,b2"}),
		Entry("service_broker_names", "service_broker_names=b1,b2", payloads.ServiceOfferingList{BrokerNames: "b1,b2"}),
		Entry("fields[service_broker]", "fields[service_broker]=guid,name", payloads.ServiceOfferingList{
			IncludeResourceRules: []params.IncludeResourceRule{{
				RelationshipPath: []string{"service_broker"},
				Fields:           []string{"guid", "name"},
			}},
		}),
	)

	DescribeTable("invalid query",
		func(query string, errMatcher types.GomegaMatcher) {
			_, decodeErr := decodeQuery[payloads.ServiceOfferingList](query)
			Expect(decodeErr).To(errMatcher)
		},
		Entry("invalid service broker field", "fields[service_broker]=foo", MatchError(ContainSubstring("value must be one of"))),
	)

	Describe("ToMessage", func() {
		It("converts payload to repository message", func() {
			payload := &payloads.ServiceOfferingList{Names: "b1,b2", BrokerNames: "br1,br2"}

			Expect(payload.ToMessage()).To(Equal(repositories.ListServiceOfferingMessage{
				Names:       []string{"b1", "b2"},
				BrokerNames: []string{"br1", "br2"},
			}))
		})
	})
})
