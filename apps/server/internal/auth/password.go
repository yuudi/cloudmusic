package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	passwordMemory      = 64 * 1024 // KiB
	passwordIterations  = 3
	passwordParallelism = 4
	passwordSaltLength  = 16
	passwordHashLength  = 32
)

var argonParameters = regexp.MustCompile(`^m=([0-9]{1,6}),t=([0-9]{1,2}),p=([0-9]{1,2})$`)

// PasswordHash stores the algorithm, version, parameters, salt and hash in PHC format.
func PasswordHash(password string) (string, error) {
	if len(password) < 8 || len(password) > 1024 {
		return "", ErrInvalid
	}
	salt, err := random(passwordSaltLength)
	if err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, passwordIterations, passwordMemory, passwordParallelism, passwordHashLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, passwordMemory, passwordIterations, passwordParallelism, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

// verifyPassword rejects unbounded parameters before allocating memory. The second
// result indicates that a successfully verified hash should be upgraded.
func verifyPassword(password, encoded string) (valid, rehash bool) {
	if len(password) > 1024 || len(encoded) > 512 {
		return false, false
	}
	parts := strings.Split(encoded, "$")
	var memory, iterations, parallelism uint32
	var saltText, hashText string
	legacy := len(parts) == 2
	if legacy {
		memory, iterations, parallelism = 64*1024, 3, 1
		saltText, hashText = parts[0], parts[1]
	} else {
		if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
			return false, false
		}
		params := argonParameters.FindStringSubmatch(parts[3])
		if params == nil {
			return false, false
		}
		m, err := strconv.ParseUint(params[1], 10, 32)
		if err != nil {
			return false, false
		}
		t, err := strconv.ParseUint(params[2], 10, 32)
		if err != nil {
			return false, false
		}
		p, err := strconv.ParseUint(params[3], 10, 8)
		if err != nil {
			return false, false
		}
		memory, iterations, parallelism = uint32(m), uint32(t), uint32(p)
		saltText, hashText = parts[4], parts[5]
	}
	// These limits prevent corrupt or hostile database values from exhausting the server.
	if memory < 8*1024 || memory > 256*1024 || iterations < 1 || iterations > 10 || parallelism < 1 || parallelism > 8 || memory < 8*parallelism {
		return false, false
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(saltText)
	if err != nil || len(salt) < 16 || len(salt) > 64 {
		return false, false
	}
	hash, err := base64.RawStdEncoding.Strict().DecodeString(hashText)
	if err != nil || len(hash) < 16 || len(hash) > 64 {
		return false, false
	}
	// The previous unversioned format always used exactly these lengths.
	if legacy && (len(salt) != 16 || len(hash) != 32) {
		return false, false
	}
	actual := argon2.IDKey([]byte(password), salt, iterations, memory, uint8(parallelism), uint32(len(hash)))
	if subtle.ConstantTimeCompare(actual, hash) != 1 {
		return false, false
	}
	// Never replace a PHC hash with lower memory or iteration settings.
	upgrade := legacy || (memory <= passwordMemory && iterations <= passwordIterations && len(hash) <= passwordHashLength && len(salt) <= passwordSaltLength &&
		(memory < passwordMemory || iterations < passwordIterations || len(hash) < passwordHashLength || len(salt) < passwordSaltLength || parallelism != passwordParallelism))
	return true, upgrade
}

func verify(password, encoded string) bool {
	valid, _ := verifyPassword(password, encoded)
	return valid
}
