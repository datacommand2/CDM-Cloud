package email

import (
	"errors"
	"net/smtp"
)

// LoginAuth login 인증
type LoginAuth struct {
	username, password string
}

// Auth login 인증을 생성한다.
func (LoginAuth) Auth(username, password, _ string) smtp.Auth {
	return &LoginAuth{username, password}
}

// Start implement smtp.auth
func (a *LoginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

// Next implement smtp.auth
func (a *LoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("unknown from server")
		}
	}
	return nil, nil
}
