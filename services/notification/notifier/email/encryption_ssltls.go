package email

import (
	"crypto/tls"
	"net"
	"net/smtp"
)

// SSLTLSEncryption ssl/tls 암호화 방식
type SSLTLSEncryption struct {
}

// Connect ssl/tls 로 연결한다.
func (SSLTLSEncryption) Connect(smtpAddress string) (*smtp.Client, error) {
	host, _, err := net.SplitHostPort(smtpAddress)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		ServerName: host,
	}
	conn, err := tls.Dial("tcp", smtpAddress, tlsConfig)
	if err != nil {
		return nil, err
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return nil, err
	}
	return c, nil
}
