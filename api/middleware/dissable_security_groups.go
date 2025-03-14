package middleware

import (
	"net/http"
	"strings"

	apierrors "code.cloudfoundry.org/korifi/api/errors"
	"code.cloudfoundry.org/korifi/api/routing"
	"github.com/go-logr/logr"
)

func DisableSecurityGroups(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logr.FromContextOrDiscard(r.Context()).WithName("disable-security-groups")

		if isSecurityGroupEndpoint(r.URL.Path) {
			routing.PresentError(logger, w, apierrors.NewInvalidRequestError(nil, "Experimental security groups support is not enabled"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isSecurityGroupEndpoint(requestPath string) bool {
	if strings.HasPrefix(requestPath, "/v3/security_groups") {
		return true
	}

	return false
}
