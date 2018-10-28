// +build ece408ProjectMode

package client

import (
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/rai-project/auth/provider"
	"github.com/rai-project/config"
	"github.com/rai-project/database/mongodb"
	"github.com/spf13/cast"
)

func (c *Client) RecordJob() error {

	if c.jobBody == nil {
		return errors.New("ranking uninitialized")
	}

	body, ok := c.jobBody.(*Ece408JobResponseBody)
	if !ok {
		panic("invalid job type")
	}

	// body.ID = ""
	body.UpdatedAt = time.Now()
	body.IsSubmission = cast.ToBool(c.options.ctx.Value(isSubmissionKey{}))

	//var s submissionKind
	//s = c.options.ctx.Value(submissionKindKey{}).(submissionKind)

	submissionKind, ok := c.options.ctx.Value(submissionKindKey{}).(submissionKind)
	if ok {
		switch submissionKind {
		case m1:
			body.SubmissionTag = "m1"
		case m2:
			body.SubmissionTag = "m2"
		case m3:
			body.SubmissionTag = "m3"
		case m4:
			body.SubmissionTag = "m4"
		case final:
			body.SubmissionTag = "final"
		case custom:
			log.Info("Using embedded eval build for custom submission")
			body.SubmissionTag = "eval"
		default:
			return errors.Errorf("unrecognized submission type %v", submissionKind)
		}
	}

	prof, err := provider.New()
	user := prof.Info()
	body.Username = user.Username
	body.UserAccessKey = user.AccessKey

	body.Teamname, err = FindTeamName(body.Username)
	if err != nil && body.IsSubmission {
		color.Red("no team name found.\n")
		body.Teamname = user.Team.Name
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
