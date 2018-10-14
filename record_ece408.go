// +build ece408ProjectMode

package client

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rai-project/auth/provider"
	"github.com/rai-project/config"
	"github.com/rai-project/database/mongodb"
	"github.com/spf13/cast"
)

func (c *Client) RecordJob() error {

	if c.job == nil {
		return errors.New("ranking uninitialized")
	}

	body, ok := c.jobBody.(*Ece408JobResponseBody)
	if !ok {
		panic("invalid job type")
	}

	defer func() {
		c.jobBody = body
	}()

	body.CreatedAt = time.Now()
	body.IsSubmission = cast.ToBool(c.options.ctx.Value(isSubmissionKey{}))
	body.SubmissionTag = cast.ToString(c.options.ctx.Value(submissionKindKey{}))

	prof, err := provider.New()
	user := prof.Info()
	body.UserID = user.ID
	body.Username = user.Username

	body.Teamname, err = FindTeamName(body.Username)
	if err != nil && body.IsSubmission {
		return errors.New("no team name found")
	}

	db, err := mongodb.NewDatabase(config.App.Name)
	if err != nil {
		return err
	}
	defer db.Close()

	col, err := NewEce408JobResponseBodyCollection(db)
	if err != nil {
		return err
	}
	defer col.Close()

	err = col.Insert(body)
	if err != nil {
		log.WithError(err).Error("Failed to insert job record:", body)
		return err
	}

	return nil
}
