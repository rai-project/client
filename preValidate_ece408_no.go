// +build !ece408ProjectMode

package client

func (c *Client) validateUserRole() error {
	role, err := c.profile.GetRole()
	if err != nil {
		return err
	}

	if isECE408Role(role) {
		return &ValidationError{
			Message: "You are using an invalid client. Please download the correct client from http://github.com/rai-project/rai",
		}
	}

	return nil
}

func (c *Client) preValidate() error {
	return nil
}
