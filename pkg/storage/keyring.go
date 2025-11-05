package storage

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the identifier used for all GoFlow credentials in the system keyring.
	ServiceName = "goflow"
)

// CredentialStore defines the interface for secure credential storage.
type CredentialStore interface {
	// Set stores a credential securely
	Set(key string, value string) error
	// Get retrieves a credential
	Get(key string) (string, error)
	// Delete removes a credential
	Delete(key string) error
	// List returns all credential keys (not the values)
	List() ([]string, error)
}

// KeyringCredentialStore implements CredentialStore using the system keyring.
// - macOS: Uses Keychain
// - Windows: Uses Credential Manager
// - Linux: Uses Secret Service (GNOME Keyring, KWallet)
type KeyringCredentialStore struct {
	service string
}

// NewKeyringCredentialStore creates a new keyring-based credential store.
func NewKeyringCredentialStore() *KeyringCredentialStore {
	return &KeyringCredentialStore{
		service: ServiceName,
	}
}

// Set stores a credential securely in the system keyring.
// The key is used as the account name, and value is the password.
func (s *KeyringCredentialStore) Set(key string, value string) error {
	if key == "" {
		return fmt.Errorf("credential key cannot be empty")
	}

	err := keyring.Set(s.service, key, value)
	if err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	// Update the credential index
	if err := s.addToIndex(key); err != nil {
		// Log warning but don't fail - credential is stored
		// In production, this would use a proper logger
		_ = err
	}

	return nil
}

// Get retrieves a credential from the system keyring.
func (s *KeyringCredentialStore) Get(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("credential key cannot be empty")
	}

	value, err := keyring.Get(s.service, key)
	if err != nil {
		// Check if error is "not found" vs other errors
		if err == keyring.ErrNotFound {
			return "", fmt.Errorf("credential not found: %s", key)
		}
		return "", fmt.Errorf("failed to retrieve credential: %w", err)
	}

	return value, nil
}

// Delete removes a credential from the system keyring.
func (s *KeyringCredentialStore) Delete(key string) error {
	if key == "" {
		return fmt.Errorf("credential key cannot be empty")
	}

	err := keyring.Delete(s.service, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return fmt.Errorf("credential not found: %s", key)
		}
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	// Update the credential index
	if err := s.removeFromIndex(key); err != nil {
		// Log warning but don't fail - credential is deleted
		_ = err
	}

	return nil
}

// List returns all credential keys stored by GoFlow.
// Note: This retrieves the credential index from the keyring.
// The index is stored as a special entry named "__goflow_index__".
func (s *KeyringCredentialStore) List() ([]string, error) {
	// Retrieve the index from keyring
	indexJSON, err := keyring.Get(s.service, "__goflow_index__")
	if err != nil {
		if err == keyring.ErrNotFound {
			// No credentials stored yet
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to retrieve credential index: %w", err)
	}

	// Parse the index
	var keys []string
	if err := json.Unmarshal([]byte(indexJSON), &keys); err != nil {
		return nil, fmt.Errorf("failed to parse credential index: %w", err)
	}

	return keys, nil
}

// addToIndex adds a key to the credential index.
func (s *KeyringCredentialStore) addToIndex(key string) error {
	// Get current index
	keys, err := s.List()
	if err != nil {
		return err
	}

	// Check if key already exists
	for _, k := range keys {
		if k == key {
			return nil // Already in index
		}
	}

	// Add new key
	keys = append(keys, key)

	// Save updated index
	return s.saveIndex(keys)
}

// removeFromIndex removes a key from the credential index.
func (s *KeyringCredentialStore) removeFromIndex(key string) error {
	// Get current index
	keys, err := s.List()
	if err != nil {
		return err
	}

	// Remove the key
	newKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != key {
			newKeys = append(newKeys, k)
		}
	}

	// Save updated index
	return s.saveIndex(newKeys)
}

// saveIndex saves the credential index to the keyring.
func (s *KeyringCredentialStore) saveIndex(keys []string) error {
	indexJSON, err := json.Marshal(keys)
	if err != nil {
		return fmt.Errorf("failed to marshal credential index: %w", err)
	}

	err = keyring.Set(s.service, "__goflow_index__", string(indexJSON))
	if err != nil {
		return fmt.Errorf("failed to save credential index: %w", err)
	}

	return nil
}

// SetStructured stores a structured credential (e.g., server config with multiple fields).
// The credential is serialized as JSON before storage.
func (s *KeyringCredentialStore) SetStructured(key string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal credential data: %w", err)
	}

	return s.Set(key, string(jsonData))
}

// GetStructured retrieves and deserializes a structured credential.
func (s *KeyringCredentialStore) GetStructured(key string, dest interface{}) error {
	jsonData, err := s.Get(key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(jsonData), dest); err != nil {
		return fmt.Errorf("failed to unmarshal credential data: %w", err)
	}

	return nil
}
