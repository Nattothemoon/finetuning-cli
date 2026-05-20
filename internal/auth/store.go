// Package auth manages the API key lifecycle:
//
//	priority on read:  --api-key flag  →  FINETUNING_API_KEY env  →  OS keychain  →  plaintext fallback file
//
// `Set` writes the keychain first, then falls back to a 0600 file at
// $config/credentials when the keychain isn't available (headless Linux,
// containers, etc.).
package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/finetuning/cli/internal/config"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "finetuning"
	keyringAccount = "default"
	envVar         = "FINETUNING_API_KEY"
)

// ErrNoKey is returned by Get when nothing is configured anywhere.
var ErrNoKey = errors.New("no API key configured (run `ft auth login`)")

// Source describes where Get found the key. Useful for `ft doctor`.
type Source string

const (
	SourceNone     Source = ""
	SourceEnv      Source = "env"
	SourceKeychain Source = "keychain"
	SourceFile     Source = "file"
)

// Get returns the active API key, consulting env → keychain → file in order.
// If `override` is non-empty, it wins.
func Get(override string) (string, Source, error) {
	if override != "" {
		return override, "flag", nil
	}
	if v := os.Getenv(envVar); v != "" {
		return v, SourceEnv, nil
	}
	if v, err := keyring.Get(keyringService, keyringAccount); err == nil && v != "" {
		return v, SourceKeychain, nil
	}
	v, err := readFallback()
	if err == nil && v != "" {
		return v, SourceFile, nil
	}
	return "", SourceNone, ErrNoKey
}

// Set stores the key. Tries keychain first; on failure, writes plaintext file (0600).
// Returns the Source that ultimately accepted the write.
func Set(key string) (Source, error) {
	if err := keyring.Set(keyringService, keyringAccount, key); err == nil {
		// Clean up any prior fallback file so we don't have two copies drifting apart.
		_ = removeFallback()
		return SourceKeychain, nil
	}
	if err := writeFallback(key); err != nil {
		return SourceNone, err
	}
	return SourceFile, nil
}

// Delete clears the key from both keychain and fallback.
func Delete() error {
	kErr := keyring.Delete(keyringService, keyringAccount)
	fErr := removeFallback()
	if kErr != nil && !errors.Is(kErr, keyring.ErrNotFound) {
		return kErr
	}
	if fErr != nil && !errors.Is(fErr, os.ErrNotExist) {
		return fErr
	}
	return nil
}

func fallbackPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials"), nil
}

func readFallback() (string, error) {
	p, err := fallbackPath()
	if err != nil {
		return "", err
	}
	buf, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	return trim(string(buf)), nil
}

func writeFallback(key string) error {
	p, err := fallbackPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	return os.WriteFile(p, []byte(key), 0o600)
}

func removeFallback() error {
	p, err := fallbackPath()
	if err != nil {
		return err
	}
	return os.Remove(p)
}

func trim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	return s
}
