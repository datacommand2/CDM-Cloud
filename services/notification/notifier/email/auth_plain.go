package email

import "net/smtp"

// PlainAuth plain 인증
type PlainAuth struct {
}

// Auth plain 인증을 생성한다.
func (PlainAuth) Auth(username, password, host string) smtp.Auth {
	return smtp.PlainAuth("", username, password, host)
}
