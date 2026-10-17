package app

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookieName = "gp_sdr_session"

// Password hashes use a portable, deliberately explicit format so the service
// never needs to keep a plaintext browser password: pbkdf2-sha256$iterations$salt$hash.
type passwordAuth struct {
	hash       string
	iterations int
	salt       []byte
	checksum   []byte
	secret     [sha256.Size]byte
}

func newPasswordAuth(encoded string) (*passwordAuth, error) {
	parts := strings.Split(strings.TrimSpace(encoded), "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return nil, errors.New("GPSDR_WEB_PASSWORD_HASH must use pbkdf2-sha256$iterations$salt$hash")
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 100000 || iterations > 2000000 {
		return nil, errors.New("GPSDR_WEB_PASSWORD_HASH iterations must be between 100000 and 2000000")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) < 16 {
		return nil, errors.New("GPSDR_WEB_PASSWORD_HASH must contain a base64 salt of at least 16 bytes")
	}
	checksum, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(checksum) != sha256.Size {
		return nil, errors.New("GPSDR_WEB_PASSWORD_HASH must contain a 32-byte base64 checksum")
	}
	return &passwordAuth{hash: encoded, iterations: iterations, salt: salt, checksum: checksum, secret: sha256.Sum256([]byte("GP-SDR password session v1\x00" + encoded))}, nil
}

func NewPasswordHash(password string) (string, error) {
	if len(password) < 12 || len(password) > 1024 {
		return "", errors.New("password must be 12 to 1024 characters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	const iterations = 310000
	checksum := pbkdf2SHA256([]byte(password), salt, iterations, sha256.Size)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iterations, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(checksum)), nil
}

func (a *passwordAuth) validPassword(password string) bool {
	checksum := pbkdf2SHA256([]byte(password), a.salt, a.iterations, len(a.checksum))
	return subtle.ConstantTimeCompare(checksum, a.checksum) == 1
}

// PBKDF2-HMAC-SHA256 is kept here rather than importing a newer standard
// package so the Pi build remains compatible with this module's Go 1.22 floor.
func pbkdf2SHA256(password, salt []byte, iterations, length int) []byte {
	output := make([]byte, 0, length)
	for block := uint32(1); len(output) < length; block++ {
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		var index [4]byte
		index[0], index[1], index[2], index[3] = byte(block>>24), byte(block>>16), byte(block>>8), byte(block)
		_, _ = mac.Write(index[:])
		u := mac.Sum(nil)
		derived := append([]byte(nil), u...)
		for round := 1; round < iterations; round++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for i := range derived {
				derived[i] ^= u[i]
			}
		}
		output = append(output, derived...)
	}
	return output[:length]
}

func (a *passwordAuth) issueSession(w http.ResponseWriter) error {
	expires := time.Now().Add(12 * time.Hour).Unix()
	nonce := make([]byte, 18)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	payload := fmt.Sprintf("%d.%s", expires, base64.RawURLEncoding.EncodeToString(nonce))
	mac := hmac.New(sha256.New, a.secret[:])
	_, _ = mac.Write([]byte(payload))
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: value, Path: "/", MaxAge: 12 * 60 * 60, HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
	return nil
}

func (a *passwordAuth) validSession(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, a.secret[:])
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return false
	}
	payloadParts := strings.Split(string(payload), ".")
	if len(payloadParts) != 2 {
		return false
	}
	expires, err := strconv.ParseInt(payloadParts[0], 10, 64)
	return err == nil && time.Now().Unix() < expires
}
