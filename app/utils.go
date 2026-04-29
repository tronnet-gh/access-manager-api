package app

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	localdb "user-manager-api/app/localdb"
	pve "user-manager-api/app/pve"
)

type Realm struct {
	Type   string
	Config any
}

type UserSession struct {
	PVE   *pve.ProxmoxClient
	Realm struct {
		Name    string
		Handler any
	}
	DB *localdb.DB
}

func GetUserSessionFromContext(c *gin.Context) (*UserSession, int, error) {
	session := sessions.Default(c)
	SessionUUID := session.Get("SessionUUID")
	if SessionUUID == nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("No auth session found")
	}
	uuid := SessionUUID.(string)
	usersession := UserSessions[uuid]
	return usersession, http.StatusOK, nil
}
