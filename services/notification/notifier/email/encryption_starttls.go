package email

import (
	"crypto/tls"
	"net"
	"net/smtp"
)

// StartTLSEncryption starttls 암호화 방식
type StartTLSEncryption struct {
}

// Connect starttls 로 연결한다.
func (StartTLSEncryption) Connect(smtpAddress string) (*smtp.Client, error) {
	host, _, err := net.SplitHostPort(smtpAddress)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{ServerName: host}

	c, err := smtp.Dial(smtpAddress)
	if err != nil {
		return nil, err
	}
	err = c.StartTLS(config)
	if err != nil {
		return nil, err
	}
	return c, nil
}
