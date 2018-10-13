package client

func (c *Client) fixDockerPushCredentials() (err error) {
	profileInfo := c.profile.Info()
	if profileInfo.DockerHub == nil {
		return
	}

	buildImage := c.buildSpec.Commands.BuildImage
	if buildImage == nil {
		return
	}
	push := buildImage.Push
	if push == nil {
		return
	}
	if !push.Push {
		return
	}
	if push.Credentials.Username == "" && push.Credentials.Password == "" {
		push.Credentials.Username = profileInfo.DockerHub.Username
		push.Credentials.Password = profileInfo.DockerHub.Password
	}
	return
}
