package client

// there are multiple place where one can specify the docker
// credentials. this canonicalized the credentials and places
// them into the spec's push credentials field
func (c *Client) fixDockerPushCredentials() (err error) {
	profileInfo := c.profile.Info()
	if profileInfo.DockerHub == nil {
		return
	}

	buildImage := c.buildSpec.Commands.BuildImage
	if buildImage == nil {
		return
	}
	// not specified that we want to push the image
	push := buildImage.Push
	if push == nil {
		return
	}
	// not pushing an image
	if !push.Push {
		return
	}
	if push.Credentials.Username == "" && push.Credentials.Password == "" {
		push.Credentials.Username = profileInfo.DockerHub.Username
		push.Credentials.Password = profileInfo.DockerHub.Password
	}
	return
}
