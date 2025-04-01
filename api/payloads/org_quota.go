package payloads

import (
	"net/url"
)

type OrgQuotasList struct {
	Names string
}

func (l OrgQuotasList) SupportedKeys() []string {
	return []string{"guids", "organization_guids", "names"}
}

func (l *OrgQuotasList) DecodeFromURLValues(values url.Values) error {
	l.Names = values.Get("names")
	return nil
}
