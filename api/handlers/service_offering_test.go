package handlers_test

import (
	"errors"
	"log"
	"net/http"
	"strings"

	apierrors "code.cloudfoundry.org/korifi/api/errors"
	. "code.cloudfoundry.org/korifi/api/handlers"
	"code.cloudfoundry.org/korifi/api/handlers/fake"
	"code.cloudfoundry.org/korifi/api/payloads"
	"code.cloudfoundry.org/korifi/api/payloads/params"
	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/api/repositories/relationships"
	"code.cloudfoundry.org/korifi/model"
	"code.cloudfoundry.org/korifi/model/services"
	. "code.cloudfoundry.org/korifi/tests/matchers"
	"code.cloudfoundry.org/korifi/tools"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServiceOffering", func() {
	var (
		requestValidator    *fake.RequestValidator
		serviceOfferingRepo *fake.CFServiceOfferingRepository
		serviceBrokerRepo   *fake.CFServiceBrokerRepository
		servicePlanRepo     *fake.CFServicePlanRepository
	)

	BeforeEach(func() {
		requestValidator = new(fake.RequestValidator)
		serviceOfferingRepo = new(fake.CFServiceOfferingRepository)
		serviceBrokerRepo = new(fake.CFServiceBrokerRepository)
		servicePlanRepo = new(fake.CFServicePlanRepository)

		apiHandler := NewServiceOffering(
			*serverURL,
			requestValidator,
			serviceOfferingRepo,
			serviceBrokerRepo,
			relationships.NewResourseRelationshipsRepo(
				serviceOfferingRepo,
				serviceBrokerRepo,
				servicePlanRepo,
			),
		)
		routerBuilder.LoadRoutes(apiHandler)
	})

	Describe("GET /v3/service_offering/:guid", func() {
		BeforeEach(func() {
			serviceOfferingRepo.GetServiceOfferingReturns(repositories.ServiceOfferingRecord{
				ServiceOffering: services.ServiceOffering{},
				CFResource: model.CFResource{
					GUID: "offering-guid",
				},
				ServiceBrokerGUID: "broker-guid",
			}, nil)
		})

		JustBeforeEach(func() {
			req, err := http.NewRequestWithContext(ctx, "GET", "/v3/service_offerings/offering-guid", nil)
			Expect(err).NotTo(HaveOccurred())

			routerBuilder.Build().ServeHTTP(rr, req)
		})

		It("returns the service offering", func() {
			Expect(serviceOfferingRepo.GetServiceOfferingCallCount()).To(Equal(1))
			_, actualAuthInfo, actualOfferingGUID := serviceOfferingRepo.GetServiceOfferingArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(actualOfferingGUID).To(Equal("offering-guid"))

			Expect(rr).Should(HaveHTTPStatus(http.StatusOK))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(rr).To(HaveHTTPBody(SatisfyAll(
				MatchJSONPath("$.guid", "offering-guid"),
			)))
		})

		When("params to inlude fields[service_broker]", func() {
			BeforeEach(func() {
				serviceBrokerRepo.ListServiceBrokersReturns([]repositories.ServiceBrokerRecord{{
					ServiceBroker: services.ServiceBroker{
						Name: "broker-name",
					},
					CFResource: model.CFResource{
						GUID: "broker-guid",
					},
				}}, nil)

				requestValidator.DecodeAndValidateURLValuesStub = decodeAndValidateURLValuesStub(&payloads.ServiceOfferingGet{
					IncludeResourceRules: []params.IncludeResourceRule{{
						RelationshipPath: []string{"service_broker"},
						Fields:           []string{"name", "guid"},
					}},
				})
			})

			It("includes service offering in the response", func() {
				log.Printf("body %+v", rr.Body)
				Expect(rr).Should(HaveHTTPStatus(http.StatusOK))
				Expect(rr).To(HaveHTTPBody(SatisfyAll(
					MatchJSONPath("$.included.service_brokers[0].name", "broker-name"),
					MatchJSONPath("$.included.service_brokers[0].guid", "broker-guid"),
				)))
			})
		})

		When("the request is invalid", func() {
			BeforeEach(func() {
				requestValidator.DecodeAndValidateURLValuesReturns(errors.New("invalid-request"))
			})

			It("returns an error", func() {
				expectUnknownError()
			})
		})

		When("getting the offering fails", func() {
			BeforeEach(func() {
				serviceOfferingRepo.GetServiceOfferingReturns(repositories.ServiceOfferingRecord{}, errors.New("get-err"))
			})

			It("returns an error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("PATCH /v3/service_offering/:guid", func() {
		BeforeEach(func() {
			serviceOfferingRepo.PatchServiceOfferingReturns(repositories.ServiceOfferingRecord{
				ServiceOffering: services.ServiceOffering{},
				CFResource: model.CFResource{
					GUID: "offering-guid",
				},
				ServiceBrokerGUID: "broker-guid",
			}, nil)

			requestValidator.DecodeAndValidateJSONPayloadStub = decodeAndValidatePayloadStub(&payloads.ServiceOfferingPatch{
				Metadata: payloads.MetadataPatch{
					Annotations: map[string]*string{"ann2": tools.PtrTo("ann_val2")},
					Labels:      map[string]*string{"lab2": tools.PtrTo("lab_val2")},
				},
			})
		})

		JustBeforeEach(func() {
			req, err := http.NewRequestWithContext(ctx, "PATCH", "/v3/service_offerings/offering-guid", strings.NewReader("the-json-body"))
			Expect(err).NotTo(HaveOccurred())
			routerBuilder.Build().ServeHTTP(rr, req)
		})

		It("patches the service offering", func() {
			Expect(requestValidator.DecodeAndValidateJSONPayloadCallCount()).To(Equal(1))

			actualReq, _ := requestValidator.DecodeAndValidateJSONPayloadArgsForCall(0)
			Expect(bodyString(actualReq)).To(Equal("the-json-body"))

			Expect(serviceOfferingRepo.GetServiceOfferingCallCount()).To(Equal(1))
			Expect(serviceOfferingRepo.PatchServiceOfferingCallCount()).To(Equal(1))
			_, actualAuthInfo, actualPatchMessage := serviceOfferingRepo.PatchServiceOfferingArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))
			Expect(actualPatchMessage).To(Equal(repositories.PatchServiceOfferingMessage{
				GUID: "offering-guid",
				Metadata: repositories.MetadataPatch{
					Annotations: map[string]*string{"ann2": tools.PtrTo("ann_val2")},
					Labels:      map[string]*string{"lab2": tools.PtrTo("lab_val2")},
				},
			}))

			Expect(rr).Should(HaveHTTPStatus(http.StatusOK))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(rr).To(HaveHTTPBody(SatisfyAll(
				MatchJSONPath("$.guid", "offering-guid"),
			)))
		})

		When("decoding the payload fails", func() {
			BeforeEach(func() {
				requestValidator.DecodeAndValidateJSONPayloadReturns(apierrors.NewUnprocessableEntityError(nil, "nope"))
			})

			It("returns an error", func() {
				expectUnprocessableEntityError("nope")
			})
		})

		When("getting the service offering fails with not found", func() {
			BeforeEach(func() {
				serviceOfferingRepo.GetServiceOfferingReturns(
					repositories.ServiceOfferingRecord{},
					apierrors.NewNotFoundError(nil, repositories.ServiceOfferingResourceType),
				)
			})

			It("returns 404 Not Found", func() {
				expectNotFoundError(repositories.ServiceOfferingResourceType)
			})
		})

		When("patching the service offering fails", func() {
			BeforeEach(func() {
				serviceOfferingRepo.PatchServiceOfferingReturns(repositories.ServiceOfferingRecord{}, errors.New("oops"))
			})

			It("returns the error", func() {
				expectUnknownError()
			})
		})
	})

	Describe("GET /v3/service_offerings", func() {
		BeforeEach(func() {
			serviceOfferingRepo.ListOfferingsReturns([]repositories.ServiceOfferingRecord{{
				ServiceOffering: services.ServiceOffering{},
				CFResource: model.CFResource{
					GUID: "offering-guid",
				},
				ServiceBrokerGUID: "broker-guid",
			}}, nil)
		})

		JustBeforeEach(func() {
			req, err := http.NewRequestWithContext(ctx, "GET", "/v3/service_offerings", nil)
			Expect(err).NotTo(HaveOccurred())

			routerBuilder.Build().ServeHTTP(rr, req)
		})

		It("lists the service offerings", func() {
			Expect(serviceOfferingRepo.ListOfferingsCallCount()).To(Equal(1))
			_, actualAuthInfo, _ := serviceOfferingRepo.ListOfferingsArgsForCall(0)
			Expect(actualAuthInfo).To(Equal(authInfo))

			Expect(rr).Should(HaveHTTPStatus(http.StatusOK))
			Expect(rr).To(HaveHTTPHeaderWithValue("Content-Type", "application/json"))
			Expect(rr).To(HaveHTTPBody(SatisfyAll(
				MatchJSONPath("$.pagination.total_results", BeEquivalentTo(1)),
				MatchJSONPath("$.pagination.first.href", "https://api.example.org/v3/service_offerings"),
				MatchJSONPath("$.resources[0].guid", "offering-guid"),
				MatchJSONPath("$.resources[0].links.self.href", "https://api.example.org/v3/service_offerings/offering-guid"),
				MatchJSONPath("$.resources[0].links.service_plans.href", "https://api.example.org/v3/service_plans?service_offering_guids=offering-guid"),
				MatchJSONPath("$.resources[0].links.service_broker.href", "https://api.example.org/v3/service_brokers/broker-guid"),
			)))
		})

		When("filtering query params are provided", func() {
			BeforeEach(func() {
				requestValidator.DecodeAndValidateURLValuesStub = decodeAndValidateURLValuesStub(&payloads.ServiceOfferingList{
					Names: "a1,a2",
				})
			})

			It("passes them to the repository", func() {
				Expect(serviceOfferingRepo.ListOfferingsCallCount()).To(Equal(1))
				_, _, message := serviceOfferingRepo.ListOfferingsArgsForCall(0)
				Expect(message.Names).To(ConsistOf("a1", "a2"))
			})
		})

		Describe("include broker fields", func() {
			BeforeEach(func() {
				serviceBrokerRepo.ListServiceBrokersReturns([]repositories.ServiceBrokerRecord{{
					ServiceBroker: services.ServiceBroker{
						Name: "broker-name",
					},
					CFResource: model.CFResource{
						GUID: "broker-guid",
					},
				}}, nil)

				requestValidator.DecodeAndValidateURLValuesStub = decodeAndValidateURLValuesStub(&payloads.ServiceOfferingList{
					IncludeResourceRules: []params.IncludeResourceRule{{
						RelationshipPath: []string{"service_broker"},
						Fields:           []string{"name", "guid"},
					}},
				})
			})

			It("lists the brokers", func() {
				Expect(serviceBrokerRepo.ListServiceBrokersCallCount()).To(Equal(1))
				_, _, actualListMessage := serviceBrokerRepo.ListServiceBrokersArgsForCall(0)
				Expect(actualListMessage).To(Equal(repositories.ListServiceBrokerMessage{
					GUIDs: []string{"broker-guid"},
				}))
			})

			When("listing brokers fails", func() {
				BeforeEach(func() {
					serviceBrokerRepo.ListServiceBrokersReturns([]repositories.ServiceBrokerRecord{}, errors.New("list-broker-err"))
				})

				It("returns an error", func() {
					expectUnknownError()
				})
			})

			Describe("broker name", func() {
				BeforeEach(func() {
					requestValidator.DecodeAndValidateURLValuesStub = decodeAndValidateURLValuesStub(&payloads.ServiceOfferingList{
						IncludeResourceRules: []params.IncludeResourceRule{{
							RelationshipPath: []string{"service_broker"},
							Fields:           []string{"name"},
						}},
					})
				})

				It("includes broker fields in the response", func() {
					Expect(rr).Should(HaveHTTPStatus(http.StatusOK))
					Expect(rr).To(HaveHTTPBody(SatisfyAll(
						MatchJSONPath("$.included.service_brokers[0].name", "broker-name"),
					)))
				})
			})

			Describe("broker guid", func() {
				BeforeEach(func() {
					requestValidator.DecodeAndValidateURLValuesStub = decodeAndValidateURLValuesStub(&payloads.ServiceOfferingList{
						IncludeResourceRules: []params.IncludeResourceRule{{
							RelationshipPath: []string{"service_broker"},
							Fields:           []string{"guid"},
						}},
					})
				})

				It("includes broker fields in the response", func() {
					Expect(rr).Should(HaveHTTPStatus(http.StatusOK))
					Expect(rr).To(HaveHTTPBody(SatisfyAll(
						MatchJSONPath("$.included.service_brokers[0].guid", "broker-guid"),
					)))
				})
			})
		})

		When("the request is invalid", func() {
			BeforeEach(func() {
				requestValidator.DecodeAndValidateURLValuesReturns(errors.New("invalid-request"))
			})

			It("returns an error", func() {
				expectUnknownError()
			})
		})

		When("listing the offerings fails", func() {
			BeforeEach(func() {
				serviceOfferingRepo.ListOfferingsReturns(nil, errors.New("list-err"))
			})

			It("returns an error", func() {
				expectUnknownError()
			})
		})
	})
})
