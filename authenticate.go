package client

import "github.com/rai-project/aws"

func (c *Client) createAWSSession() error {

	// Create an AWS session
	session, err := aws.NewSession(
		aws.Region(aws.AWSRegionUSEast1),
		aws.AccessKey(aws.Config.AccessKey),
		aws.SecretKey(aws.Config.SecretKey),
		aws.Sts(c.ID.Hex()),
	)
	if err != nil {
		return err
	}
	c.awsSession = session
	return nil
}

func (c *Client) Authenticate() error {
	if err := c.createAWSSession(); err != nil {
		return err
	}

	if err := c.fixDockerPushCredentials(); err != nil {
		return err
	}

	return nil
}
