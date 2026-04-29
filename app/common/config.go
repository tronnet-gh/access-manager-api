package app

import (
	"encoding/json"
	"os"
)

type PVEConfig struct {
	URL   string `json:"url"`
	Token struct {
		User  string `json:"user"`
		Realm string `json:"realm"`
		ID    string `json:"id"`
		UUID  string `json:"uuid"`
	} `json:"token"`
	PAASClientRole string `json:"paas-client-role"`
}

type LDAPConfig struct {
	BaseDN   string `json:""`
	LdapURL  string
	StartTLS bool
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
	PVE PVEConfig `json:"pve"`
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
