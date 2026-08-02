package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

const (
	argonMemory      = 19 * 1024
	argonIterations  = 2
	argonParallelism = 1
	argonSaltLength  = 16
	argonKeyLength   = 32
)

func normalizeEmail(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func validateEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	address, err := mail.ParseAddress(value)

	if err != nil || address.Address != value || !strings.Contains(value, "@") {
		return "", errors.New("enter a valid email address")
	}

	return value, nil
}

func validatePassword(password string) (string, error) {
	password = norm.NFC.String(password)
	length := utf8.RuneCountInString(password)

	if length < 8 || length > 128 {
		return "", errors.New("password must be between 8 and 128 characters")
	}

	return password, nil
}

func hashPassword(password string) (string, error) {
	normalized, err := validatePassword(password)
	if err != nil {
		return "", err
	}

	//random salt for every password
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey(
		[]byte(normalized),
		salt,
		argonIterations,
		argonMemory,
		argonParallelism,
		argonKeyLength,
	)

	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func verifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}

	var memory uint64
	var iterations uint64
	var parallelism uint64

	for item := range strings.SplitSeq(parts[3], ",") {
		pair := strings.SplitN(item, "=", 2)
		if len(pair) != 2 {
			return false
		}

		value, err := strconv.ParseUint(pair[1], 10, 32)
		if err != nil {
			return false
		}

		switch pair[0] {
		case "m":
			memory = value
		case "t":
			iterations = value
		case "p":
			parallelism = value
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	normalized := norm.NFC.String(password)
	actual := argon2.IDKey([]byte(normalized), salt, uint32(iterations), uint32(memory), uint8(parallelism), uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
