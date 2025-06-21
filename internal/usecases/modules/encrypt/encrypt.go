package encrypt

import "golang.org/x/crypto/bcrypt"

func Encrypt(password string) (string, error) {
	salt, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(salt), nil
}
