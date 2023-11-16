package email

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// AuthDAO auth 방식에 따른 처리를 한다.
type AuthDAO interface {
	Auth(username, password, host string) smtp.Auth
}

// EncryptionDAO encryption 방식에 따른 연결을 처리한다.
type EncryptionDAO interface {
	Connect(smtpAddress string) (*smtp.Client, error)
}

// Email 메일을 보내기 위한 절차를 수행한다.
type Email struct {
	auth       AuthDAO
	encryption EncryptionDAO
	client     *smtp.Client
	username   string
	password   string
	host       string
}

// NewEmail encryption 및 auth 방식에 따른 새로운 Email 객체를 생성한다.
func NewEmail(encryption, auth string) (*Email, error) {
	var (
		authDao       AuthDAO
		encryptionDao EncryptionDAO
	)

	switch strings.ToLower(encryption) {
	case "ssl/tls":
		encryptionDao = SSLTLSEncryption{}
	case "starttls":
		encryptionDao = StartTLSEncryption{}
	default:
		return nil, UnsupportedEncryption(encryption)
	}

	switch strings.ToLower(auth) {
	case "plain":
		authDao = &PlainAuth{}
	case "login":
		authDao = &LoginAuth{}
	case "cram-md5":
		authDao = &CramMD5Auth{}
	default:
		return nil, UnsupportedAuth(auth)
	}

	return &Email{
		auth:       authDao,
		encryption: encryptionDao,
	}, nil
}

// Connect SMTP 서버에 연결한다.
func (e *Email) Connect(smtpAddress, username, password string) error {
	var err error
	e.username = username
	e.password = password

	host, _, err := net.SplitHostPort(smtpAddress)
	if err != nil {
		return err
	}
	e.host = host

	e.client, err = e.encryption.Connect(smtpAddress)
	if err != nil {
		return err
	}

	return nil
}

// Auth 서버에 인증요청을 한다.
func (e *Email) Auth() error {
	auth := e.auth.Auth(e.username, e.password, e.host)
	err := e.client.Auth(auth)
	if err != nil {
		return err
	}

	return e.client.Mail(e.username)
}

// Send 상대방에게 메일을 보낸다.
func (e *Email) Send(to, subject, body string) error {
	err := e.client.Rcpt(to)
	if err != nil {
		return err
	}

	w, err := e.client.Data()
	if err != nil {
		return err
	}

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n",
		e.username, to, subject, body)

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}

	return w.Close()
}

// Close SMTP 연결을 종료한다.
func (e *Email) Close() error {
	return e.client.Quit()
}
