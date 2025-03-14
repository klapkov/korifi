package handlers_test

import (
	"errors"
	"net/http"
	"strings"

	apierrors "code.cloudfoundry.org/korifi/api/errors"
	. "code.cloudfoundry.org/korifi/api/handlers"
	"code.cloudfoundry.org/korifi/api/handlers/fake"
	"code.cloudfoundry.org/korifi/api/payloads"
	"code.cloudfoundry.org/korifi/api/repositories"
	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	. "code.cloudfoundry.org/korifi/tests/matchers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SecurityGroup", func() {
	var (
		requestMethod     string
		requestPath       string
		requestBody       string
		securityGroupRepo *fake.CFSecurityGroupRepository
		spaceRepo         *fake.CFSpaceRepository
		requestValidator  *fake.RequestValidator
	)

	BeforeEach(func() {
		securityGroupRepo = new(fake.CFSecurityGroupRepository)
		spaceRepo = new(fake.CFSpaceRepository)
		requestValidator = new(fake.RequestValidator)

		apiHandler := NewSecurityGroup(
			*serverURL,
			securityGroupRepo,
			spaceRepo,
			requestValidator,
		)
		routerBuilder.LoadRoutes(apiHandler)
	})

	JustBeforeEach(func() {
		req, err := http.NewRequestWithContext(ctx, requestMethod, requestPath, strings.NewReader(requestBody))
		Expect(err).NotTo(HaveOccurred())

		routerBuilder.Build().ServeHTTP(rr, req)
	})

	Describe("GET /v3/security_groups/{guid}", func() {
		BeforeEach(func() {
			requestMethod = http.MethodGet
			requestPath = "/v3/security_groups/test-guid"
			requestBody = ""

			securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
				Name: "test-security-group",
			}, nil)
		})

		It("returns the security group", func() {
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(rr).To(HaveHTTPBody(SatisfyAll(
				MatchJSONPath("$.guid", "test-guid"),
				MatchJSONPath("$.name", "test-security-group"),
			)))
		})

		When("the security group is not found", func() {
			BeforeEach(func() {
				securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{}, apierrors.NewNotFoundError(nil, "SecurityGroup"))
			})

			It("returns a not found error", func() {
				expectNotFoundError("SecurityGroup")
			})
		})

		When("the repository returns an error", func() {
			BeforeEach(func() {
				securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{}, errors.New("repo-error"))
			})

			It("returns an unknown error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("POST /v3/security_groups", func() {
		var payload payloads.SecurityGroupCreate

		BeforeEach(func() {
			requestMethod = http.MethodPost
			requestPath = "/v3/security_groups"
			requestBody = "the-json-body"

			payload = payloads.SecurityGroupCreate{
				DisplayName: "test-security-group",
				Rules: []korifiv1alpha1.SecurityGroupRule{
					{
						Protocol:    korifiv1alpha1.ProtocolTCP,
						Ports:       "80",
						Destination: "192.168.1.1",
					},
				},
				Relationships: payloads.SecurityGroupRelationships{
					RunningSpaces: payloads.ToManyRelationship{
						Data: []payloads.RelationshipData{
							{
								GUID: "space1",
							},
						},
					},
					StagingSpaces: payloads.ToManyRelationship{
						Data: []payloads.RelationshipData{
							{
								GUID: "space2",
							},
						},
					},
				},
			}
			requestValidator.DecodeAndValidateJSONPayloadStub = decodeAndValidatePayloadStub(&payload)

			securityGroupRepo.CreateSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
				Name: "test-security-group",
				Rules: []korifiv1alpha1.SecurityGroupRule{
					{
						Protocol:    korifiv1alpha1.ProtocolTCP,
						Ports:       "80",
						Destination: "192.168.1.1",
					},
				},
				RunningSpaces: []string{"space1"},
				StagingSpaces: []string{"space2"},
			}, nil)
		})

		It("validates the request", func() {
			Expect(requestValidator.DecodeAndValidateJSONPayloadCallCount()).To(Equal(1))
			actualReq, _ := requestValidator.DecodeAndValidateJSONPayloadArgsForCall(0)
			Expect(bodyString(actualReq)).To(Equal("the-json-body"))
		})

		It("creates a security group with a rule", func() {
			Expect(securityGroupRepo.CreateSecurityGroupCallCount()).To(Equal(1))
			Expect(spaceRepo.ListSpacesCallCount()).To(Equal(1))

			_, actualAuthInfo, createSecurityGroupMessage := securityGroupRepo.CreateSecurityGroupArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(createSecurityGroupMessage.DisplayName).To(Equal("test-security-group"))
			Expect(createSecurityGroupMessage.Rules).To(Equal([]korifiv1alpha1.SecurityGroupRule{
				{
					Protocol:    korifiv1alpha1.ProtocolTCP,
					Ports:       "80",
					Destination: "192.168.1.1",
				},
			}))

			_, _, listSpacesMessage := spaceRepo.ListSpacesArgsForCall(0)
			Expect(listSpacesMessage.GUIDs).To(ConsistOf("space1", "space2"))

			Expect(rr).To(HaveHTTPStatus(http.StatusCreated))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(rr).To(HaveHTTPBody(SatisfyAll(
				MatchJSONPath("$.guid", "test-guid"),
				MatchJSONPath("$.name", "test-security-group"),
				MatchJSONPath("$.rules[0].protocol", "tcp"),
				MatchJSONPath("$.rules[0].ports", "80"),
				MatchJSONPath("$.rules[0].destination", "192.168.1.1"),
				MatchJSONPath("$.relationships.running_spaces.data[0].guid", "space1"),
				MatchJSONPath("$.relationships.staging_spaces.data[0].guid", "space2"),
			)))
		})

		When("the requested binded space does not exist", func() {
			BeforeEach(func() {
				spaceRepo.ListSpacesReturns([]repositories.SpaceRecord{}, errors.New("boom"))
			})

			It("returns an error", func() {
				expectUnknownError()
			})
		})

		When("the request body is not valid", func() {
			BeforeEach(func() {
				requestValidator.DecodeAndValidateJSONPayloadReturns(apierrors.NewUnprocessableEntityError(nil, "nope"))
			})

			It("returns an error", func() {
				expectUnprocessableEntityError("nope")
			})
		})

		When("the repository returns an error", func() {
			BeforeEach(func() {
				securityGroupRepo.CreateSecurityGroupReturns(repositories.SecurityGroupRecord{}, errors.New("repo-error"))
			})

			It("returns an unknown error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("GET /v3/security_groups", func() {
		BeforeEach(func() {
			requestMethod = http.MethodGet
			requestPath = "/v3/security_groups?names=test-security-group"

			payload := payloads.SecurityGroupList{Names: "test-security-group"}
			requestValidator.DecodeAndValidateURLValuesStub = decodeAndValidateURLValuesStub(&payload)

			securityGroupRepo.ListSecurityGroupsReturns([]repositories.SecurityGroupRecord{
				{GUID: "test-guid", Name: "test-security-group"},
			}, nil)
		})

		It("lists the security groups", func() {
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(rr).To(HaveHTTPBody(SatisfyAll(
				MatchJSONPath("$.resources[0].guid", "test-guid"),
				MatchJSONPath("$.resources[0].name", "test-security-group"),
			)))

			Expect(securityGroupRepo.ListSecurityGroupsCallCount()).To(Equal(1))
			_, actualAuthInfo, listMessage := securityGroupRepo.ListSecurityGroupsArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(listMessage.Names).To(ConsistOf("test-security-group"))
		})

		When("the query parameters are invalid", func() {
			BeforeEach(func() {
				requestValidator.DecodeAndValidateURLValuesReturns(errors.New("validation-error"))
			})

			It("returns a validation error", func() {
				expectUnprocessableEntityError("validation-error")
			})
		})

		When("the repository returns an error", func() {
			BeforeEach(func() {
				securityGroupRepo.ListSecurityGroupsReturns(nil, errors.New("repo-error"))
			})

			It("returns an unknown error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("PATCH /v3/security_groups/{guid}", func() {
		var payload payloads.SecurityGroupUpdate

		BeforeEach(func() {
			requestMethod = http.MethodPatch
			requestPath = "/v3/security_groups/test-guid"
			requestBody = `{"name": "updated-security-group"}`

			payload = payloads.SecurityGroupUpdate{DisplayName: "updated-security-group"}
			requestValidator.DecodeAndValidateJSONPayloadStub = decodeAndValidatePayloadStub(&payload)

			securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
				Name: "test-security-group",
			}, nil)

			securityGroupRepo.UpdateSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
				Name: "updated-security-group",
			}, nil)
		})

		It("updates the security group", func() {
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(rr).To(HaveHTTPBody(SatisfyAll(
				MatchJSONPath("$.guid", "test-guid"),
				MatchJSONPath("$.name", "updated-security-group"),
			)))

			Expect(securityGroupRepo.UpdateSecurityGroupCallCount()).To(Equal(1))
			_, actualAuthInfo, updateMessage := securityGroupRepo.UpdateSecurityGroupArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(updateMessage.GUID).To(Equal("test-guid"))
			Expect(updateMessage.DisplayName).To(Equal("updated-security-group"))
		})

		When("the payload is invalid", func() {
			BeforeEach(func() {
				requestValidator.DecodeAndValidateJSONPayloadReturns(errors.New("validation-error"))
			})

			It("returns a validation error", func() {
				expectUnprocessableEntityError("validation-error")
			})
		})

		When("the security group does not exist", func() {
			BeforeEach(func() {
				securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{}, apierrors.NewNotFoundError(nil, "SecurityGroup"))
			})

			It("returns a not found error", func() {
				expectNotFoundError("SecurityGroup")
			})
		})

		When("the repository returns an error", func() {
			BeforeEach(func() {
				securityGroupRepo.UpdateSecurityGroupReturns(repositories.SecurityGroupRecord{}, errors.New("repo-error"))
			})

			It("returns an unknown error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("POST /v3/security_groups/{guid}/relationships/running_spaces", func() {
		var payload payloads.SecurityGroupBindRunning

		BeforeEach(func() {
			requestMethod = http.MethodPost
			requestPath = "/v3/security_groups/test-guid/relationships/running_spaces"
			requestBody = `{"data": [{"guid": "space-guid"}]}`

			payload = payloads.SecurityGroupBindRunning{
				Data: []payloads.RelationshipData{{GUID: "space-guid"}},
			}
			requestValidator.DecodeAndValidateJSONPayloadStub = decodeAndValidatePayloadStub(&payload)

			securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
			}, nil)

			spaceRepo.ListSpacesReturns([]repositories.SpaceRecord{
				{GUID: "space-guid"},
			}, nil)

			securityGroupRepo.BindRunningSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
			}, nil)
		})

		It("binds running spaces to the security group", func() {
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))

			Expect(securityGroupRepo.BindRunningSecurityGroupCallCount()).To(Equal(1))
			_, actualAuthInfo, bindMessage := securityGroupRepo.BindRunningSecurityGroupArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(bindMessage.GUID).To(Equal("test-guid"))
			Expect(bindMessage.Spaces).To(ConsistOf("space-guid"))
		})

		When("the payload is invalid", func() {
			BeforeEach(func() {
				requestValidator.DecodeAndValidateJSONPayloadReturns(errors.New("validation-error"))
			})

			It("returns a validation error", func() {
				expectUnprocessableEntityError("validation-error")
			})
		})

		When("the security group does not exist", func() {
			BeforeEach(func() {
				securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{}, apierrors.NewNotFoundError(nil, "SecurityGroup"))
			})

			It("returns a not found error", func() {
				expectNotFoundError("SecurityGroup")
			})
		})

		When("the space does not exist", func() {
			BeforeEach(func() {
				spaceRepo.ListSpacesReturns([]repositories.SpaceRecord{}, nil)
			})

			It("returns an unprocessable entity error", func() {
				expectUnprocessableEntityError("failed to bind security group, space  does not exist")
			})
		})

		When("the repository returns an error", func() {
			BeforeEach(func() {
				securityGroupRepo.BindRunningSecurityGroupReturns(repositories.SecurityGroupRecord{}, errors.New("repo-error"))
			})

			It("returns an unknown error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("POST /v3/security_groups/{guid}/relationships/staging_spaces", func() {
		var payload payloads.SecurityGroupBindStaging

		BeforeEach(func() {
			requestMethod = http.MethodPost
			requestPath = "/v3/security_groups/test-guid/relationships/staging_spaces"
			requestBody = `{"data": [{"guid": "space-guid"}]}`

			payload = payloads.SecurityGroupBindStaging{
				Data: []payloads.RelationshipData{{GUID: "space-guid"}},
			}
			requestValidator.DecodeAndValidateJSONPayloadStub = decodeAndValidatePayloadStub(&payload)

			securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
			}, nil)

			spaceRepo.ListSpacesReturns([]repositories.SpaceRecord{
				{GUID: "space-guid"},
			}, nil)

			securityGroupRepo.BindStagingSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
			}, nil)
		})

		It("binds staging spaces to the security group", func() {
			Expect(rr).To(HaveHTTPStatus(http.StatusOK))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))

			Expect(securityGroupRepo.BindStagingSecurityGroupCallCount()).To(Equal(1))
			_, actualAuthInfo, bindMessage := securityGroupRepo.BindStagingSecurityGroupArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(bindMessage.GUID).To(Equal("test-guid"))
			Expect(bindMessage.Spaces).To(ConsistOf("space-guid"))
		})

		When("the payload is invalid", func() {
			BeforeEach(func() {
				requestValidator.DecodeAndValidateJSONPayloadReturns(errors.New("validation-error"))
			})

			It("returns a validation error", func() {
				expectUnprocessableEntityError("validation-error")
			})
		})

		When("the security group does not exist", func() {
			BeforeEach(func() {
				securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{}, apierrors.NewNotFoundError(nil, "SecurityGroup"))
			})

			It("returns a not found error", func() {
				expectNotFoundError("SecurityGroup")
			})
		})

		When("the space does not exist", func() {
			BeforeEach(func() {
				spaceRepo.ListSpacesReturns([]repositories.SpaceRecord{}, nil)
			})

			It("returns an unprocessable entity error", func() {
				expectUnprocessableEntityError("failed to bind security group, space  does not exist")
			})
		})

		When("the repository returns an error", func() {
			BeforeEach(func() {
				securityGroupRepo.BindStagingSecurityGroupReturns(repositories.SecurityGroupRecord{}, errors.New("repo-error"))
			})

			It("returns an unknown error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("DELETE /v3/security_groups/{guid}/relationships/running_spaces/{space_guid}", func() {
		BeforeEach(func() {
			requestMethod = http.MethodDelete
			requestPath = "/v3/security_groups/test-guid/relationships/running_spaces/space-guid"

			securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
			}, nil)

			spaceRepo.GetSpaceReturns(repositories.SpaceRecord{
				GUID: "space-guid",
			}, nil)

			securityGroupRepo.UnbindRunningSecurityGroupReturns(nil)
		})

		It("unbinds the running space from the security group", func() {
			Expect(rr).To(HaveHTTPStatus(http.StatusNoContent))

			Expect(securityGroupRepo.UnbindRunningSecurityGroupCallCount()).To(Equal(1))
			_, actualAuthInfo, unbindMessage := securityGroupRepo.UnbindRunningSecurityGroupArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(unbindMessage.GUID).To(Equal("test-guid"))
			Expect(unbindMessage.SpaceGUID).To(Equal("space-guid"))
		})

		When("the security group does not exist", func() {
			BeforeEach(func() {
				securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{}, apierrors.NewNotFoundError(nil, "SecurityGroup"))
			})

			It("returns a not found error", func() {
				expectNotFoundError("SecurityGroup")
			})
		})

		When("the space does not exist", func() {
			BeforeEach(func() {
				spaceRepo.GetSpaceReturns(repositories.SpaceRecord{}, apierrors.NewNotFoundError(nil, "Space"))
			})

			It("returns a not found error", func() {
				expectNotFoundError("Space")
			})
		})

		When("the repository returns an error", func() {
			BeforeEach(func() {
				securityGroupRepo.UnbindRunningSecurityGroupReturns(errors.New("repo-error"))
			})

			It("returns an unknown error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("DELETE /v3/security_groups/{guid}/relationships/staging_spaces/{space_guid}", func() {
		BeforeEach(func() {
			requestMethod = http.MethodDelete
			requestPath = "/v3/security_groups/test-guid/relationships/staging_spaces/space-guid"

			securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{
				GUID: "test-guid",
			}, nil)

			spaceRepo.GetSpaceReturns(repositories.SpaceRecord{
				GUID: "space-guid",
			}, nil)

			securityGroupRepo.UnbindStagingSecurityGroupReturns(nil)
		})

		It("unbinds the staging space from the security group", func() {
			Expect(rr).To(HaveHTTPStatus(http.StatusNoContent))

			Expect(securityGroupRepo.UnbindStagingSecurityGroupCallCount()).To(Equal(1))
			_, actualAuthInfo, unbindMessage := securityGroupRepo.UnbindStagingSecurityGroupArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(unbindMessage.GUID).To(Equal("test-guid"))
			Expect(unbindMessage.SpaceGUID).To(Equal("space-guid"))
		})

		When("the security group does not exist", func() {
			BeforeEach(func() {
				securityGroupRepo.GetSecurityGroupReturns(repositories.SecurityGroupRecord{}, apierrors.NewNotFoundError(nil, "SecurityGroup"))
			})

			It("returns a not found error", func() {
				expectNotFoundError("SecurityGroup")
			})
		})

		When("the space does not exist", func() {
			BeforeEach(func() {
				spaceRepo.GetSpaceReturns(repositories.SpaceRecord{}, apierrors.NewNotFoundError(nil, "Space"))
			})

			It("returns a not found error", func() {
				expectNotFoundError("Space")
			})
		})

		When("the repository returns an error", func() {
			BeforeEach(func() {
				securityGroupRepo.UnbindStagingSecurityGroupReturns(errors.New("repo-error"))
			})

			It("returns an unknown error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("DELETE /v3/security_groups/{guid}", func() {
		BeforeEach(func() {
			requestMethod = http.MethodDelete
			requestPath = "/v3/security_groups/test-guid"

			securityGroupRepo.DeleteSecurityGroupReturns(nil)
		})

		It("deletes the security group", func() {
			Expect(rr).To(HaveHTTPStatus(http.StatusNoContent))

			Expect(securityGroupRepo.DeleteSecurityGroupCallCount()).To(Equal(1))
			_, actualAuthInfo, guid := securityGroupRepo.DeleteSecurityGroupArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(guid).To(Equal("test-guid"))
		})

		When("the repository returns an error", func() {
			BeforeEach(func() {
				securityGroupRepo.DeleteSecurityGroupReturns(errors.New("repo-error"))
			})

			It("returns an unknown error", func() {
				expectUnknownError()
			})
		})
	})
})
