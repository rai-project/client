package client

import "time"

// SubmitRecord describes a submission in MongoDB
type SubmitRecord struct {
	TeamName string
	Time     time.Time
}
