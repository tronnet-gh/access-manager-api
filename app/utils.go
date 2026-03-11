package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	ldap "user-manager-api/app/ldap"
	"user-manager-api/app/pve"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	common "user-manager-api/app/common"
)

type Backends struct {
	handler string
	pve     *pve.ProxmoxClient
	ldap    *ldap.LDAPClient
}

func GetBackendsFromContext(c *gin.Context) (*Backends, int, error) {
	session := sessions.Default(c)
	SessionUUID := session.Get("SessionUUID")
	if SessionUUID == nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("No auth session found")
	}
	uuid := SessionUUID.(string)
	usersession := UserSessions[uuid]
	return usersession, http.StatusOK, nil
}

func LoadLocaldb(dbPath string) (map[string]common.User, error) {
	users := map[string]common.User{}
	content, err := os.ReadFile(dbPath)
	if err != nil {
		//log.Fatal("Error when opening file: ", err)
		return users, err
	}
	err = json.Unmarshal(content, &users)
	if err != nil {
		//log.Fatal("Error during Unmarshal(): ", err)
		return users, err
	}
	return users, nil
}

func SaveLocaldb(configPath string, users map[string]common.User) error {
	json, err := json.Marshal(users)
	if err != nil {
		return err
	}
	err = os.WriteFile(configPath, []byte(json), 0644)
	return err
}
