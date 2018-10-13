// +build !ece408ProjectMode

package client

func (c *Client) validateUserRole() {
  role := "" // todo: get role

  if role == "ece408_student" {
    panic(ValidationError{
      Message: "You are using an invalid client. Please download the correct client from http://github.com/rai-project/rai"
    })
  }
}
