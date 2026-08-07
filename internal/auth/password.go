package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Time      uint32 = 1
	argon2Memory    uint32 = 64 * 1024
	argon2Threads   uint8  = 2
	argon2KeyLength        = 32
	saltLength             = 16
)

// HashPassword hashes plain with argon2id and returns a PHC-style string.
func HashPassword(plain string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(plain), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLength)

	return fmt.Sprintf(
		"argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// CheckPassword reports whether plain matches the encoded argon2id hash.
func CheckPassword(hash, plain string) bool {
	params, salt, expected, err := parsePasswordHash(hash)
	if err != nil {
		return false
	}

	actual := argon2.IDKey([]byte(plain), salt, params.time, params.memory, params.threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

type passwordParams struct {
	time    uint32
	memory  uint32
	threads uint8
}

func parsePasswordHash(hash string) (passwordParams, []byte, []byte, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 5 {
		return passwordParams{}, nil, nil, errors.New("invalid password hash format")
	}
	if parts[0] != "argon2id" {
		return passwordParams{}, nil, nil, errors.New("unsupported password hash algorithm")
	}
	if parts[1] != "v=19" {
		return passwordParams{}, nil, nil, errors.New("unsupported password hash version")
	}

	params, err := parsePasswordParams(parts[2])
	if err != nil {
		return passwordParams{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return passwordParams{}, nil, nil, fmt.Errorf("decode salt: %w", err)
	}

	expected, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return passwordParams{}, nil, nil, fmt.Errorf("decode hash: %w", err)
	}

	return params, salt, expected, nil
}

func parsePasswordParams(value string) (passwordParams, error) {
	fields := strings.Split(value, ",")
	if len(fields) != 3 {
		return passwordParams{}, errors.New("invalid password hash parameters")
	}

	memory, err := parseUintField(fields[0], "m")
	if err != nil {
		return passwordParams{}, err
	}
	timeCost, err := parseUintField(fields[1], "t")
	if err != nil {
		return passwordParams{}, err
	}
	threads, err := parseUintField(fields[2], "p")
	if err != nil {
		return passwordParams{}, err
	}
	if threads > 255 {
		return passwordParams{}, errors.New("invalid password hash thread count")
	}

	return passwordParams{
		memory:  uint32(memory),
		time:    uint32(timeCost),
		threads: uint8(threads),
	}, nil
}

func parseUintField(value, prefix string) (uint64, error) {
	raw, ok := strings.CutPrefix(value, prefix+"=")
	if !ok {
		return 0, fmt.Errorf("invalid password hash parameter %q", prefix)
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse password hash parameter %q: %w", prefix, err)
	}
	return n, nil
}
