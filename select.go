package client

import "github.com/rai-project/config"

func (c *Client) selectBuildQueue() {
	c.buildFileJobQueueName = config.App.Name + "_" + c.buildSpec.Resources.CPU.Architecture
	log.Debug("inferring queue ", c.buildFileJobQueueName, " from build file. May be overridden by client.Options")
}
