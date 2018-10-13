package client

import (
	"io/ioutil"

	"github.com/pkg/errors"
)

func (c *Client) Validate() error {

	options := c.options

	// Authenticate user
	if err := c.authenticate(options.profilePath); err != nil {
		return err
	}

	if err := c.preValidate(); err != nil {
		return err
	}

	specFilePath, err := c.findSpecFile()
	if err != nil {
		return err
	}

	buf, err := ioutil.ReadFile(specFilePath)
	if err != nil {
		return errors.Wrapf(err, "unable to read %v", specFilePath)
	}

	if err := c.readSpec(buf); err != nil {
		return err
	}
	return nil
}
