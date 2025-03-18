package repositories

import "time"

type SpaceQuotaRecord struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	GUID      string
}

type SpaceData struct {
	GUID string
}
