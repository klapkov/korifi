package repositories

import "time"

type OrgQuotaRecord struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	GUID      string
}

type OrgData struct {
	GUID string
}
