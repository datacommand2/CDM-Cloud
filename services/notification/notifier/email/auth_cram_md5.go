package email

import "net/smtp"

// CramMD5Auth CRAM-MD5 인증
type CramMD5Auth struct {
}

// Auth CRAM-MD5 인증을 생성한다.
func (CramMD5Auth) Auth(username, password, _ string) smtp.Auth {
	return smtp.CRAMMD5Auth(username, password)
}
