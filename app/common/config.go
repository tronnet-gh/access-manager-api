package app

import (
	"encoding/json"
	"log"
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

func GetConfig(configPath string) Config {
	root, err := os.OpenRoot(".")
	if err != nil {
		log.Fatal("Error when opening root dir: ", err)
	}
	defer root.Close()

	content, err := root.ReadFile(configPath)
	if err != nil {
		log.Fatal("Error when opening config file: ", err)
	}

	var config Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		log.Fatal("Error during parsing config file: ", err)
	}

	return config
}
