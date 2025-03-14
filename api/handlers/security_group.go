package handlers

import (
	"context"
	"net/http"
	"net/url"

	"code.cloudfoundry.org/korifi/api/authorization"
	apierrors "code.cloudfoundry.org/korifi/api/errors"
	"code.cloudfoundry.org/korifi/api/payloads"
	"code.cloudfoundry.org/korifi/api/presenter"
	"code.cloudfoundry.org/korifi/api/repositories"
	"code.cloudfoundry.org/korifi/api/routing"
	"github.com/go-logr/logr"
)

const (
	SecurityGroupsPath                   = "/v3/security_groups"
	SecurityGroupPath                    = "/v3/security_groups/{guid}"
	SecurityGroupRunningSpacesPath       = "/v3/security_groups/{guid}/relationships/running_spaces"
	SecurityGroupStagingSpacesPath       = "/v3/security_groups/{guid}/relationships/staging_spaces"
	UnbindSecurityGroupRunningSpacesPath = "/v3/security_groups/{guid}/relationships/running_spaces/{space_guid}"
	UnbindSecurityGroupStagingSpacesPath = "/v3/security_groups/{guid}/relationships/staging_spaces/{space_guid}"
)

type SecurityGroup struct {
	serverURL         url.URL
	securityGroupRepo CFSecurityGroupRepository
	spaceRepo         CFSpaceRepository
	requestValidator  RequestValidator
}

//counterfeiter:generate -o fake -fake-name CFSecurityGroupRepository . CFSecurityGroupRepository
type CFSecurityGroupRepository interface {
	GetSecurityGroup(context.Context, authorization.Info, string) (repositories.SecurityGroupRecord, error)
	CreateSecurityGroup(context.Context, authorization.Info, repositories.CreateSecurityGroupMessage) (repositories.SecurityGroupRecord, error)
	ListSecurityGroups(context.Context, authorization.Info, repositories.ListSecurityGroupMessage) ([]repositories.SecurityGroupRecord, error)
	UpdateSecurityGroup(context.Context, authorization.Info, repositories.UpdateSecurityGroupMessage) (repositories.SecurityGroupRecord, error)
	BindRunningSecurityGroup(context.Context, authorization.Info, repositories.BindRunningSecurityGroupMessage) (repositories.SecurityGroupRecord, error)
	BindStagingSecurityGroup(context.Context, authorization.Info, repositories.BindStagingSecurityGroupMessage) (repositories.SecurityGroupRecord, error)
	UnbindRunningSecurityGroup(context.Context, authorization.Info, repositories.UnbindRunningSecurityGroupMessage) error
	UnbindStagingSecurityGroup(context.Context, authorization.Info, repositories.UnbindStagingSecurityGroupMessage) error
	DeleteSecurityGroup(context.Context, authorization.Info, string) error
}

func NewSecurityGroup(
	serverURL url.URL,
	securityGroupRepo CFSecurityGroupRepository,
	spaceRepo CFSpaceRepository,
	requestValidator RequestValidator,
) *SecurityGroup {
	return &SecurityGroup{
		serverURL:         serverURL,
		securityGroupRepo: securityGroupRepo,
		spaceRepo:         spaceRepo,
		requestValidator:  requestValidator,
	}
}

func (h *SecurityGroup) get(r *http.Request) (*routing.Response, error) {
	authInfo, _ := authorization.InfoFromContext(r.Context())
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.security-group.get")

	guid := routing.URLParam(r, "guid")
	securityGroup, err := h.securityGroupRepo.GetSecurityGroup(r.Context(), authInfo, guid)
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to get security group")
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForSecurityGroup(securityGroup, h.serverURL)), nil
}

func (h *SecurityGroup) create(r *http.Request) (*routing.Response, error) {
	authInfo, _ := authorization.InfoFromContext(r.Context())
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.security-group.create")

	payload := new(payloads.SecurityGroupCreate)
	if err := h.requestValidator.DecodeAndValidateJSONPayload(r, payload); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to decode payload")
	}

	securityGroup, err := h.securityGroupRepo.CreateSecurityGroup(r.Context(), authInfo, payload.ToMessage())
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to create security group")
	}

	_, err = h.spaceRepo.ListSpaces(r.Context(), authInfo, repositories.ListSpacesMessage{
		GUIDs: append(payload.Relationships.RunningSpaces.CollectGUIDs(),
			payload.Relationships.StagingSpaces.CollectGUIDs()...),
	})
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to create security group, space  does not exist")
	}

	return routing.NewResponse(http.StatusCreated).WithBody(presenter.ForSecurityGroup(securityGroup, h.serverURL)), nil
}

func (h *SecurityGroup) list(r *http.Request) (*routing.Response, error) {
	authInfo, _ := authorization.InfoFromContext(r.Context())
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.security-group.list")

	payload := new(payloads.SecurityGroupList)
	if err := h.requestValidator.DecodeAndValidateURLValues(r, payload); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to decode payload")
	}

	securityGroups, err := h.securityGroupRepo.ListSecurityGroups(r.Context(), authInfo, payload.ToMessage())
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to list security groups")
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForList(presenter.ForSecurityGroup, securityGroups, h.serverURL, *r.URL)), nil
}

func (h *SecurityGroup) update(r *http.Request) (*routing.Response, error) {
	authInfo, _ := authorization.InfoFromContext(r.Context())
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.security-group.update")

	payload := new(payloads.SecurityGroupUpdate)
	if err := h.requestValidator.DecodeAndValidateJSONPayload(r, payload); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to decode payload")
	}

	guid := routing.URLParam(r, "guid")
	_, err := h.securityGroupRepo.GetSecurityGroup(r.Context(), authInfo, guid)
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to update security group, it does not exist")
	}

	securityGroup, err := h.securityGroupRepo.UpdateSecurityGroup(r.Context(), authInfo, payload.ToMessage(guid))
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to update security group")
	}

	return routing.NewResponse(http.StatusCreated).WithBody(presenter.ForSecurityGroup(securityGroup, h.serverURL)), nil
}

func (h *SecurityGroup) bindRunning(r *http.Request) (*routing.Response, error) {
	authInfo, _ := authorization.InfoFromContext(r.Context())
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.security-group.bind-running-spaces")

	payload := new(payloads.SecurityGroupBindRunning)
	if err := h.requestValidator.DecodeAndValidateJSONPayload(r, payload); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to decode payload")
	}

	guid := routing.URLParam(r, "guid")
	_, err := h.securityGroupRepo.GetSecurityGroup(r.Context(), authInfo, guid)
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group, it does not exist")
	}

	var spaceGUIDs []string
	for _, space := range payload.Data {
		spaceGUIDs = append(spaceGUIDs, space.GUID)
	}

	if _, err = h.spaceRepo.ListSpaces(r.Context(), authInfo, repositories.ListSpacesMessage{GUIDs: spaceGUIDs}); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group, space  does not exist")
	}

	securityGroup, err := h.securityGroupRepo.BindRunningSecurityGroup(r.Context(), authInfo, payload.ToMessage(guid))
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group to running space")
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForSecurityGroupRunningSpaces(securityGroup, h.serverURL)), nil
}

func (h *SecurityGroup) bindStaging(r *http.Request) (*routing.Response, error) {
	authInfo, _ := authorization.InfoFromContext(r.Context())
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.security-group.bind-staging-spaces")

	payload := new(payloads.SecurityGroupBindStaging)
	if err := h.requestValidator.DecodeAndValidateJSONPayload(r, payload); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to decode payload")
	}

	guid := routing.URLParam(r, "guid")
	_, err := h.securityGroupRepo.GetSecurityGroup(r.Context(), authInfo, guid)
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group, it does not exist")
	}

	var spaceGUIDs []string
	for _, space := range payload.Data {
		spaceGUIDs = append(spaceGUIDs, space.GUID)
	}

	if _, err = h.spaceRepo.ListSpaces(r.Context(), authInfo, repositories.ListSpacesMessage{GUIDs: spaceGUIDs}); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group, space  does not exist")
	}

	securityGroup, err := h.securityGroupRepo.BindStagingSecurityGroup(r.Context(), authInfo, payload.ToMessage(guid))
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group to running space")
	}

	return routing.NewResponse(http.StatusOK).WithBody(presenter.ForSecurityGroupStagingSpaces(securityGroup, h.serverURL)), nil
}

func (h *SecurityGroup) unbindRunning(r *http.Request) (*routing.Response, error) {
	authInfo, _ := authorization.InfoFromContext(r.Context())
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.security-group.unbind-running-spaces")

	guid := routing.URLParam(r, "guid")
	spaceGuid := routing.URLParam(r, "space_guid")

	_, err := h.securityGroupRepo.GetSecurityGroup(r.Context(), authInfo, guid)
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group, it does not exist")
	}

	if _, err = h.spaceRepo.GetSpace(r.Context(), authInfo, spaceGuid); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group, space  does not exist")
	}

	if err := h.securityGroupRepo.UnbindRunningSecurityGroup(r.Context(), authInfo, repositories.UnbindRunningSecurityGroupMessage{
		GUID:      guid,
		SpaceGUID: spaceGuid,
	}); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to unbind security group to running space")
	}

	return routing.NewResponse(http.StatusNoContent), nil
}

func (h *SecurityGroup) unbindStaging(r *http.Request) (*routing.Response, error) {
	authInfo, _ := authorization.InfoFromContext(r.Context())
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.security-group.unbind-staging-spaces")

	guid := routing.URLParam(r, "guid")
	spaceGuid := routing.URLParam(r, "space_guid")

	_, err := h.securityGroupRepo.GetSecurityGroup(r.Context(), authInfo, guid)
	if err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group, it does not exist")
	}

	if _, err = h.spaceRepo.GetSpace(r.Context(), authInfo, spaceGuid); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to bind security group, space  does not exist")
	}

	if err := h.securityGroupRepo.UnbindStagingSecurityGroup(r.Context(), authInfo, repositories.UnbindStagingSecurityGroupMessage{
		GUID:      guid,
		SpaceGUID: spaceGuid,
	}); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to unbind security group to staging space")
	}

	return routing.NewResponse(http.StatusNoContent), nil
}

func (h *SecurityGroup) delete(r *http.Request) (*routing.Response, error) {
	authInfo, _ := authorization.InfoFromContext(r.Context())
	logger := logr.FromContextOrDiscard(r.Context()).WithName("handlers.security-group.delete")

	guid := routing.URLParam(r, "guid")
	if err := h.securityGroupRepo.DeleteSecurityGroup(r.Context(), authInfo, guid); err != nil {
		return nil, apierrors.LogAndReturn(logger, err, "failed to delete security group with guid: %s", guid)
	}

	return routing.NewResponse(http.StatusNoContent), nil
}

func (h *SecurityGroup) UnauthenticatedRoutes() []routing.Route {
	return nil
}

func (h *SecurityGroup) AuthenticatedRoutes() []routing.Route {
	return []routing.Route{
		{Method: "GET", Pattern: SecurityGroupPath, Handler: h.get},
		{Method: "POST", Pattern: SecurityGroupsPath, Handler: h.create},
		{Method: "GET", Pattern: SecurityGroupsPath, Handler: h.list},
		{Method: "PATCH", Pattern: SecurityGroupPath, Handler: h.update},
		{Method: "POST", Pattern: SecurityGroupRunningSpacesPath, Handler: h.bindRunning},
		{Method: "POST", Pattern: SecurityGroupStagingSpacesPath, Handler: h.bindStaging},
		{Method: "DELETE", Pattern: UnbindSecurityGroupRunningSpacesPath, Handler: h.unbindRunning},
		{Method: "DELETE", Pattern: UnbindSecurityGroupStagingSpacesPath, Handler: h.unbindStaging},
		{Method: "DELETE", Pattern: SecurityGroupPath, Handler: h.delete},
	}
}
