package app

import (
	"encoding/json"
	"os"
)

type LDAPConfig struct {
	LdapURL  string `json:"ldapURL"`
	StartTLS bool   `json:"startTLS"`
	BaseDN   string `json:"baseDN"`
}

type PVEConfig struct {
	URL string `json:"url"`
}

type RealmConfig struct {
	Handler string `json:"handler"`
}

type Config struct {
	ListenPort        int    `json:"listenPort"`
	SessionCookieName string `json:"sessionCookieName"`
	SessionCookie     struct {
		Path     string `json:"path"`
		HttpOnly bool   `json:"httpOnly"`
		Secure   bool   `json:"secure"`
		MaxAge   int    `json:"maxAge"`
	}
	LDAP   LDAPConfig             `json:"ldap"`
	PVE    PVEConfig              `json:"pve"`
	Realms map[string]RealmConfig `json:"realms"`
}

func GetConfig(configPath string) (Config, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}
	var config Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		return Config{}, err
	}
	return config, nil
}
