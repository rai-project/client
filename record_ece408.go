// +build ece408ProjectMode

package client

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rai-project/auth/provider"
	"github.com/rai-project/config"
	"github.com/rai-project/database/mongodb"
	"github.com/rai-project/model"
)

func (c *Client) RecordJob() error {

	if c.job == nil {
		return errors.New("ranking uninitialized")
	}

	c.job.CreatedAt = time.Now()
	c.job.IsSubmission = c.options.isSubmission
	if c.options.submissionKind != custom {
		c.job.SubmissionTag = string(c.options.submissionKind)
	} else {
		c.job.SubmissionTag = c.options.customSubmissionTag
	}

	prof, err := provider.New()
	user := prof.Info()
	c.job.Username = user.Username

	log.Debug("Submission username: " + c.job.Username)

	c.job.Teamname, err = TeamName(c.job.Username)

	log.Debug("Submission teamname: " + c.job.Teamname)

	if c.job.IsSubmission {
		if c.job.Teamname == "" {
			return errors.New("no team name found")
		}
	}

	db, err := mongodb.NewDatabase(config.App.Name)
	if err != nil {
		return err
	}
	defer db.Close()

	log.Debug("Connecting to table: rankings")

	col, err := model.NewECE408ResponseBodyCollection(db)
	if err != nil {
		return err
	}
	defer col.Close()

	err = col.Insert(c.job)
	if err != nil {
		log.WithError(err).Error("Failed to insert job record:", c.job)
	}

	return err
}
