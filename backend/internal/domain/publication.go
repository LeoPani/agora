package domain

import (
	"encoding/json"
	"time"
)

type Publication struct {
	ID              int64
	OpenAlexID      string
	DOI             string
	Title           string
	Abstract        string
	PublicationYear int
	PublicationDate *time.Time
	Type            string
	CitedByCount    int
	Topics          json.RawMessage
	CreatedAt       time.Time
}

type PublicationAuthor struct {
	PublicationID    int64
	ResearcherID     int64
	AuthorPosition   int
	IsCorresponding  bool
}

type CollectorRun struct {
	ID               int64
	CollectorName    string
	StartedAt        time.Time
	FinishedAt       *time.Time
	Status           string
	RecordsCollected int
	ErrorMessage     string
}
