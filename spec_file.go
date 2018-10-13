package client

import (
	"path/filepath"

	"github.com/Unknwon/com"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
)

func (c *Client) readSpec(buf []byte) error {
	if err := yaml.Unmarshal(buf, &c.buildSpec); err != nil {
		return errors.Wrapf(err, "unable to parse build file")
	}

	return nil
}

func (c *Client) validateSpecPermissions() error {
	return nil
}

func (c *Client) findSpecFile() (string, error) {

	options := c.options

	var buildFilePath string
	if options.buildFilePath == "" {
		buildFilePath = filepath.Join(options.directory, options.buildFileBaseName+".yml")
	} else {
		buildFilePath = options.buildFilePath
	}
	if !com.IsFile(buildFilePath) {
		return "", errors.Errorf("the spec file [%v] does not exist", buildFilePath)
	}
	if loc, err := filepath.Abs(buildFilePath); err == nil {
		buildFilePath = loc
	}

	return buildFilePath, nil
}
