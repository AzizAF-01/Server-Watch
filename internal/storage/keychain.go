package storage

import (
	"sync"

	"github.com/zalando/go-keyring"
)

const keyringServiceName = "ServerWatch"

// KeychainManager provides access to OS credential store (Windows Credential Manager / macOS Keychain / libsecret)
type KeychainManager struct {
	mu       sync.RWMutex
	fallback map[string]string // in-memory fallback if OS keyring is unavailable
}

func NewKeychainManager() *KeychainManager {
	return &KeychainManager{
		fallback: make(map[string]string),
	}
}

func (k *KeychainManager) SetPassword(serverID string, password string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	err := keyring.Set(keyringServiceName, "pwd_"+serverID, password)
	if err != nil {
		k.fallback["pwd_"+serverID] = password
	}
	return nil
}

func (k *KeychainManager) GetPassword(serverID string) (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	val, err := keyring.Get(keyringServiceName, "pwd_"+serverID)
	if err == nil {
		return val, nil
	}
	if v, ok := k.fallback["pwd_"+serverID]; ok {
		return v, nil
	}
	return "", err
}

func (k *KeychainManager) SetPassphrase(serverID string, passphrase string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	err := keyring.Set(keyringServiceName, "passphrase_"+serverID, passphrase)
	if err != nil {
		k.fallback["passphrase_"+serverID] = passphrase
	}
	return nil
}

func (k *KeychainManager) GetPassphrase(serverID string) (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	val, err := keyring.Get(keyringServiceName, "passphrase_"+serverID)
	if err == nil {
		return val, nil
	}
	if v, ok := k.fallback["passphrase_"+serverID]; ok {
		return v, nil
	}
	return "", err
}

func (k *KeychainManager) DeleteCredentials(serverID string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	_ = keyring.Delete(keyringServiceName, "pwd_"+serverID)
	_ = keyring.Delete(keyringServiceName, "passphrase_"+serverID)
	delete(k.fallback, "pwd_"+serverID)
	delete(k.fallback, "passphrase_"+serverID)
}

func (k *KeychainManager) SetDBPassword(dbConfigID string, password string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	err := keyring.Set(keyringServiceName, "db_pwd_"+dbConfigID, password)
	if err != nil {
		k.fallback["db_pwd_"+dbConfigID] = password
	}
	return nil
}

func (k *KeychainManager) GetDBPassword(dbConfigID string) (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	val, err := keyring.Get(keyringServiceName, "db_pwd_"+dbConfigID)
	if err == nil {
		return val, nil
	}
	if v, ok := k.fallback["db_pwd_"+dbConfigID]; ok {
		return v, nil
	}
	return "", err
}

func (k *KeychainManager) DeleteDBCredentials(dbConfigID string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	_ = keyring.Delete(keyringServiceName, "db_pwd_"+dbConfigID)
	delete(k.fallback, "db_pwd_"+dbConfigID)
}

func (k *KeychainManager) SetNotificationSecret(key string, secret string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	err := keyring.Set(keyringServiceName, "notif_"+key, secret)
	if err != nil {
		k.fallback["notif_"+key] = secret
	}
	return nil
}

func (k *KeychainManager) GetNotificationSecret(key string) (string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	val, err := keyring.Get(keyringServiceName, "notif_"+key)
	if err == nil {
		return val, nil
	}
	if v, ok := k.fallback["notif_"+key]; ok {
		return v, nil
	}
	return "", err
}

func (k *KeychainManager) DeleteNotificationSecret(key string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	_ = keyring.Delete(keyringServiceName, "notif_"+key)
	delete(k.fallback, "notif_"+key)
}

