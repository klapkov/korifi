package repositories

import "time"

type IsolationSegmentRecord struct {
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	GUID      string
}
