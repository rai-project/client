// +build !ece408ProjectMode

package client

func (c *Client) validateUserRole() error {
	role := "" // todo: get role

	if role == "ece408_student" {
		return &ValidationError{
			Message: "You are using an invalid client. Please download the correct client from http://github.com/rai-project/rai",
		}
	}

	return nil
}

func (c *Client) preValidate() error {
	return nil
}
