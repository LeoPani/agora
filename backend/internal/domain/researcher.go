package domain

import "time"

type Researcher struct {
	ID             int64
	OpenAlexID     string
	ORCID          string
	FullName       string
	NormalizedName string
	Department     string
	Institution    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
