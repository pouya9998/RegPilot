package tests

import "github.com/regpilot/regpilot/internal/utils"

func registryEncrypt(plain, key string) (string, error) {
	return utils.Encrypt(plain, key)
}

func registryDecrypt(enc, key string) (string, error) {
	return utils.Decrypt(enc, key)
}
