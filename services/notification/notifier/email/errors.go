package email

import "github.com/datacommand2/cdm-cloud/common/errors"

var (
	//ErrUnsupportedEncryption 지원하지 않는 encryption
	ErrUnsupportedEncryption = errors.New("unsupported encryption")
	//ErrUnsupportedAuth 지원하지 않는 auth
	ErrUnsupportedAuth = errors.New("unsupported auth")
)

// UnsupportedEncryption 지원하지 않는 encryption
func UnsupportedEncryption(enc string) error {
	return errors.Wrap(
		ErrUnsupportedEncryption,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"enc": enc,
		}),
	)
}

// UnsupportedAuth 지원하지 않는 auth
func UnsupportedAuth(auth string) error {
	return errors.Wrap(
		ErrUnsupportedAuth,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"auth": auth,
		}),
	)
}
