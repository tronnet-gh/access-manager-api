package app

import (
	"encoding/json"
	"log"
	"os"
)

type PVEConfig struct {
	URL            string      `json:"url"`
	Token          PVEAPIToken `json:"token"`
	PAASClientRole string      `json:"paas-client-role"`
}

type LDAPConfig struct {
	BaseDN   string `json:""`
	Hostname string
	StartTLS bool
	TLS      bool
	Verify   bool
}

type LocalDBConfig struct {
	Path string `json:"path"`
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
	PVE     PVEConfig     `json:"pve"`
	LocalDB LocalDBConfig `json:"localdb"`
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
