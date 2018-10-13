// +build ece408ProjectMode

package client

func (c *Client) validateUserRole() {
  role := "" // todo: get role

  if role != "ece408_student" {
    panic(ValidationError{
      Message: "You are using an invalid client. Please download the correct client from http://github.com/rai-project/rai"
    })
  }
}

func (c *Client) validateSubmission() error {
	var buf []byte


  options := c.options

  switch options.submissionKind {
  case m1:
    buf = m1Build
  case m2:
    buf = m2Build
  case m3:
    buf = m3Build
  case m4:
    buf = m4Build
  case final:
    buf = finalBuild
  case custom:
    log.Info("Using embedded eval build for custom submission")
    buf = evalBuild
  default:
    return errors.New("unrecognized submission type " + string(options.submissionKind))
  }
  fprintf(c.options.stdout, color.YellowString("✱ Using the following build file for submission:\n%s"), string(buf))

  if err := readSpec(buf); err != nil {
    return err
  }

  for _, requiredFileName := range Config.SubmitRequirements {
    requiredFilePath := filepath.Join(options.directory, requiredFileName)
    if !com.IsFile(requiredFilePath) {
      return errors.Errorf("Didn't find a file required for submission: [%v]", requiredFilePath)
    }
  }

	// Fail early on no connection to submission database
  db, err := mongodb.NewDatabase(config.App.Name)
  if err != nil {
    log.WithError(err).Error("Unable to contact submission database")
    return err
  }
  defer db.Close()

  return nil
}

// Validate ...
func (c *Client) preValidate() error {
	if options.isSubmission {
    if err := c.validateSubmission(); err != nil {
      return err
    }
  }

	return nil
}
