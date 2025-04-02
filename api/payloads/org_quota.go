package payloads

import (
	"net/url"
	"regexp"
)

type OrgQuotasList struct {
	Names string `json:"names"`
}

func (l OrgQuotasList) SupportedKeys() []string {
	return []string{"guids", "organization_guids", "names", "order_by"}
}

func (d *OrgQuotasList) IgnoredKeys() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile("page"),
		regexp.MustCompile("per_page"),
	}
}

func (l *OrgQuotasList) DecodeFromURLValues(values url.Values) error {
	l.Names = values.Get("names")
	return nil
}
