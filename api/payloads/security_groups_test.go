package payloads_test

import (
	"code.cloudfoundry.org/korifi/api/payloads"
	"code.cloudfoundry.org/korifi/api/repositories"
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/tools"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("SecurityGroupCreate", func() {
	var (
		createPayload       payloads.SecurityGroupCreate
		securityGroupCreate *payloads.SecurityGroupCreate
		validatorErr        error
	)

	BeforeEach(func() {
		securityGroupCreate = new(payloads.SecurityGroupCreate)
	})

	FDescribe("Validation", func() {
		BeforeEach(func() {
			createPayload = payloads.SecurityGroupCreate{
				DisplayName: "test-security-group",
				Rules: []korifiv1alpha1.SecurityGroupRule{
					{
						Protocol:    korifiv1alpha1.ProtocolTCP,
						Ports:       "80",
						Destination: "192.168.1.1",
					},
				},
				GloballyEnabled: korifiv1alpha1.SecurityGroupWorkloads{
					Running: false,
					Staging: false,
				},
				Relationships: payloads.SecurityGroupRelationships{
					RunningSpaces: payloads.ToManyRelationship{Data: []payloads.RelationshipData{{GUID: "space1"}}},
					StagingSpaces: payloads.ToManyRelationship{Data: []payloads.RelationshipData{{GUID: "space2"}}},
				},
			}
		})

		JustBeforeEach(func() {
			validatorErr = validator.DecodeAndValidateJSONPayload(createJSONRequest(createPayload), securityGroupCreate)
		})

		It("succeeds with valid payload", func() {
			Expect(validatorErr).NotTo(HaveOccurred())
			Expect(securityGroupCreate).To(PointTo(Equal(createPayload)))
		})

		When("The display name is empty", func() {
			BeforeEach(func() {
				createPayload.DisplayName = ""
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "name cannot be blank")
			})
		})

		When("The rules are empty", func() {
			BeforeEach(func() {
				createPayload.Rules = []korifiv1alpha1.SecurityGroupRule{}
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "rules cannot be blank")
			})
		})

		When("The protocol is invalid", func() {
			BeforeEach(func() {
				createPayload.Rules[0].Protocol = "invalid"
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "Rules[0]: protocol invalid not supported")
			})
		})

		When("Protocol is ALL with ports", func() {
			BeforeEach(func() {
				createPayload.Rules[0].Protocol = korifiv1alpha1.ProtocolALL
				createPayload.Rules[0].Ports = "80"
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "Rules[0]: ports are not allowed for protocols of type all")
			})
		})

		When("Protocol is TCP but has no ports", func() {
			BeforeEach(func() {
				createPayload.Rules[0].Protocol = korifiv1alpha1.ProtocolTCP
				createPayload.Rules[0].Ports = ""
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "Rules[0]: ports are required for protocols of type TCP and UDP")
			})
		})

		When("Destination is invalid", func() {
			BeforeEach(func() {
				createPayload.Rules[0].Destination = "invalid-dest"
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "Rules[0]: The Destination: invalid-dest is not in a valid format")
			})
		})

		When("Ports are invalid", func() {
			BeforeEach(func() {
				createPayload.Rules[0].Ports = "invalid-port"
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "Rules[0]: The ports: invalid-port is not in a valid format")
			})
		})
	})

	Describe("ToMessage", func() {
		var message repositories.CreateSecurityGroupMessage

		BeforeEach(func() {
			createPayload = payloads.SecurityGroupCreate{
				DisplayName: "test-security-group",
				Rules: []korifiv1alpha1.SecurityGroupRule{
					{Protocol: korifiv1alpha1.ProtocolTCP, Ports: "80", Destination: "192.168.1.1"},
				},
				GloballyEnabled: korifiv1alpha1.SecurityGroupWorkloads{Running: false, Staging: false},
				Relationships: payloads.SecurityGroupRelationships{
					RunningSpaces: payloads.ToManyRelationship{Data: []payloads.RelationshipData{{GUID: "space1"}}},
					StagingSpaces: payloads.ToManyRelationship{Data: []payloads.RelationshipData{{GUID: "space2"}}},
				},
			}
		})

		JustBeforeEach(func() {
			message = createPayload.ToMessage()
		})

		It("converts to repo message correctly", func() {
			Expect(message.DisplayName).To(Equal("test-security-group"))
			Expect(message.Rules).To(Equal(createPayload.Rules))
			Expect(message.GloballyEnabled).To(Equal(korifiv1alpha1.SecurityGroupWorkloads{Running: false, Staging: false}))
			Expect(message.Spaces).To(MatchAllKeys(Keys{
				"space1": Equal(korifiv1alpha1.SecurityGroupWorkloads{Running: true}),
				"space2": Equal(korifiv1alpha1.SecurityGroupWorkloads{Staging: true}),
			}))
		})
	})
})

var _ = Describe("SecurityGroupList", func() {
	DescribeTable("valid query",
		func(query string, expectedSecurityGroupList payloads.SecurityGroupList) {
			actualSecurityGroupList, decodeErr := decodeQuery[payloads.SecurityGroupList](query)
			Expect(decodeErr).NotTo(HaveOccurred())
			Expect(*actualSecurityGroupList).To(Equal(expectedSecurityGroupList))
		},
		Entry("guids", "guids=guid1,guid2", payloads.SecurityGroupList{GUIDs: "guid1,guid2"}),
		Entry("names", "names=name1,name2", payloads.SecurityGroupList{Names: "name1,name2"}),
		Entry("globally_enabled_running", "globally_enabled_running=true", payloads.SecurityGroupList{GloballyEnabledRunning: tools.PtrTo(true)}),
		Entry("globally_enabled_staging", "globally_enabled_staging=true", payloads.SecurityGroupList{GloballyEnabledStaging: tools.PtrTo(true)}),
		Entry("running_space_guids", "running_space_guids=guid1,guid2", payloads.SecurityGroupList{RunningSpaceGUIDs: "guid1,guid2"}),
		Entry("staging_space_guids", "staging_space_guids=guid1,guid2", payloads.SecurityGroupList{StagingSpaceGUIDs: "guid1,guid2"}),
	)

	DescribeTable("invalid query",
		func(query string, expectedErrMsg string) {
			_, decodeErr := decodeQuery[payloads.SecurityGroupList](query)
			Expect(decodeErr).To(MatchError(ContainSubstring(expectedErrMsg)))
		},
		Entry("unsupported param", "foo=bar", "unsupported query parameter: foo"),
		Entry("invalid globally_enabled_running", "globally_enabled_running=invalid", "failed to parse 'globally_enabled_running' query parameter"),
		Entry("invalid globally_enabled_staging", "globally_enabled_staging=invalid", "failed to parse 'globally_enabled_staging' query parameter"),
	)

	Describe("ToMessage", func() {
		var (
			payload payloads.SecurityGroupList
			message repositories.ListSecurityGroupMessage
		)

		BeforeEach(func() {
			payload = payloads.SecurityGroupList{
				GUIDs:                  "g1,g2",
				Names:                  "n1,n2",
				GloballyEnabledRunning: tools.PtrTo(true),
				GloballyEnabledStaging: tools.PtrTo(false),
				RunningSpaceGUIDs:      "rs1,rs2",
				StagingSpaceGUIDs:      "ss1,ss2",
			}
		})

		JustBeforeEach(func() {
			message = payload.ToMessage()
		})

		It("converts to repo message correctly", func() {
			Expect(message).To(Equal(repositories.ListSecurityGroupMessage{
				GUIDs:                  []string{"g1", "g2"},
				Names:                  []string{"n1", "n2"},
				GloballyEnabledRunning: tools.PtrTo(true),
				GloballyEnabledStaging: tools.PtrTo(false),
				RunningSpaceGUIDs:      []string{"rs1", "rs2"},
				StagingSpaceGUIDs:      []string{"ss1", "ss2"},
			}))
		})
	})
})

var _ = Describe("SecurityGroupUpdate", func() {
	var (
		updatePayload       payloads.SecurityGroupUpdate
		securityGroupUpdate *payloads.SecurityGroupUpdate
		validatorErr        error
	)

	BeforeEach(func() {
		securityGroupUpdate = new(payloads.SecurityGroupUpdate)
	})

	Describe("Validation", func() {
		BeforeEach(func() {
			updatePayload = payloads.SecurityGroupUpdate{
				DisplayName: "updated-security-group",
				Rules: []korifiv1alpha1.SecurityGroupRule{
					{Protocol: korifiv1alpha1.ProtocolUDP, Ports: "443", Destination: "10.0.0.0/24"},
				},
				GloballyEnabled: korifiv1alpha1.GloballyEnabledUpdate{Running: tools.PtrTo(true)},
			}
		})

		JustBeforeEach(func() {
			validatorErr = validator.DecodeAndValidateJSONPayload(createJSONRequest(updatePayload), securityGroupUpdate)
		})

		It("succeeds with valid payload", func() {
			Expect(validatorErr).NotTo(HaveOccurred())
			Expect(securityGroupUpdate).To(PointTo(Equal(updatePayload)))
		})

		When("Rules have invalid protocol", func() {
			BeforeEach(func() {
				updatePayload.Rules[0].Protocol = "INVALID"
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "Rules[0]: protocol INVALID not supported")
			})
		})

		When("Rules have invalid ports", func() {
			BeforeEach(func() {
				updatePayload.Rules[0].Ports = "invalid"
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "Rules[0]: invalid port: invalid")
			})
		})
	})

	Describe("ToMessage", func() {
		var message repositories.UpdateSecurityGroupMessage

		BeforeEach(func() {
			updatePayload = payloads.SecurityGroupUpdate{
				DisplayName: "updated-security-group",
				Rules: []korifiv1alpha1.SecurityGroupRule{
					{Protocol: korifiv1alpha1.ProtocolUDP, Ports: "443", Destination: "10.0.0.0/24"},
				},
				GloballyEnabled: korifiv1alpha1.GloballyEnabledUpdate{Running: tools.PtrTo(true)},
			}
		})

		JustBeforeEach(func() {
			message = updatePayload.ToMessage("test-guid")
		})

		It("converts to repo message correctly", func() {
			Expect(message).To(Equal(repositories.UpdateSecurityGroupMessage{
				GUID:            "test-guid",
				DisplayName:     "updated-security-group",
				Rules:           updatePayload.Rules,
				GloballyEnabled: korifiv1alpha1.GloballyEnabledUpdate{Running: tools.PtrTo(true)},
			}))
		})
	})
})

var _ = Describe("SecurityGroupBindRunning", func() {
	var (
		bindPayload       payloads.SecurityGroupBindRunning
		securityGroupBind *payloads.SecurityGroupBindRunning
		validatorErr      error
	)

	BeforeEach(func() {
		securityGroupBind = new(payloads.SecurityGroupBindRunning)
	})

	Describe("Validation", func() {
		BeforeEach(func() {
			bindPayload = payloads.SecurityGroupBindRunning{
				Data: []payloads.RelationshipData{{GUID: "space1"}, {GUID: "space2"}},
			}
		})

		JustBeforeEach(func() {
			validatorErr = validator.DecodeAndValidateJSONPayload(createJSONRequest(bindPayload), securityGroupBind)
		})

		It("succeeds with valid payload", func() {
			Expect(validatorErr).NotTo(HaveOccurred())
			Expect(securityGroupBind).To(PointTo(Equal(bindPayload)))
		})

		When("Data is empty", func() {
			BeforeEach(func() {
				bindPayload.Data = []payloads.RelationshipData{}
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "data cannot be blank")
			})
		})
	})

	Describe("ToMessage", func() {
		var message repositories.BindRunningSecurityGroupMessage

		BeforeEach(func() {
			bindPayload = payloads.SecurityGroupBindRunning{
				Data: []payloads.RelationshipData{{GUID: "space1"}, {GUID: "space2"}},
			}
		})

		JustBeforeEach(func() {
			message = bindPayload.ToMessage("sg-guid")
		})

		It("converts to repo message correctly", func() {
			Expect(message).To(Equal(repositories.BindRunningSecurityGroupMessage{
				GUID:   "sg-guid",
				Spaces: []string{"space1", "space2"},
			}))
		})
	})
})

var _ = Describe("SecurityGroupBindStaging", func() {
	var (
		bindPayload       payloads.SecurityGroupBindStaging
		securityGroupBind *payloads.SecurityGroupBindStaging
		validatorErr      error
	)

	BeforeEach(func() {
		securityGroupBind = new(payloads.SecurityGroupBindStaging)
	})

	Describe("Validation", func() {
		BeforeEach(func() {
			bindPayload = payloads.SecurityGroupBindStaging{
				Data: []payloads.RelationshipData{{GUID: "space1"}, {GUID: "space2"}},
			}
		})

		JustBeforeEach(func() {
			validatorErr = validator.DecodeAndValidateJSONPayload(createJSONRequest(bindPayload), securityGroupBind)
		})

		It("succeeds with valid payload", func() {
			Expect(validatorErr).NotTo(HaveOccurred())
			Expect(securityGroupBind).To(PointTo(Equal(bindPayload)))
		})

		When("Data is empty", func() {
			BeforeEach(func() {
				bindPayload.Data = []payloads.RelationshipData{}
			})
			It("returns an error", func() {
				expectUnprocessableEntityError(validatorErr, "data cannot be blank")
			})
		})
	})

	Describe("ToMessage", func() {
		var message repositories.BindStagingSecurityGroupMessage

		BeforeEach(func() {
			bindPayload = payloads.SecurityGroupBindStaging{
				Data: []payloads.RelationshipData{{GUID: "space1"}, {GUID: "space2"}},
			}
		})

		JustBeforeEach(func() {
			message = bindPayload.ToMessage("sg-guid")
		})

		It("converts to repo message correctly", func() {
			Expect(message).To(Equal(repositories.BindStagingSecurityGroupMessage{
				GUID:   "sg-guid",
				Spaces: []string{"space1", "space2"},
			}))
		})
	})
})
