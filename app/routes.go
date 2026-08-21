package app

import (
	common "access-manager-api/app/common"
	ldap "access-manager-api/app/ldap"
	"access-manager-api/app/localdb"
	pve "access-manager-api/app/pve"
	"fmt"
	"net/http"
	paas "proxmoxaas-common-lib"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	uuid "github.com/nu7hatch/gouuid"
)

func POST_Ticket(c *gin.Context) {
	body := common.Login{}
	err := c.ShouldBindJSON(&body)
	if err != nil { // bad request from binding
		c.JSON(http.StatusBadRequest, gin.H{"auth": false, "error": err.Error()})
		return
	}

	// attempt to parse username
	body.Username, err = paas.ParseUsername(body.UsernameRaw)
	if err != nil { // username format incorrect
		c.JSON(http.StatusBadRequest, gin.H{"auth": false, "error": err.Error()})
		return
	}
	handler := Realms[body.Username.Realm].Type

	userbackends := UserSession{}

	// always bind proxmox backend
	PVEClient, code, err := pve.NewClientFromCredentials(Config.PVE, body.Username, body.Password)
	if err != nil { // pve client failed to bind
		c.JSON(code, gin.H{"auth": false, "error": err.Error()})
		return
	}
	userbackends.PVE = PVEClient

	// bind backend by type
	switch handler {
	case "pve":
	case "ldap":
		config := Realms[body.Username.Realm].Config.(common.LDAPConfig)
		LDAPClient, code, err := ldap.NewClientFromCredentials(config, body.Username, body.Password)
		if err != nil { // ldap client failed to bind
			c.JSON(code, gin.H{"auth": false, "error": err.Error()})
			return
		}
		userbackends.Realm.Name = body.Username.Realm
		userbackends.Realm.Handler = LDAPClient
	default:
		c.JSON(code, gin.H{"auth": false, "error": fmt.Errorf("user realm %s is not supported", body.Username.Realm)})
		return
	}

	// failing to load lcoaldb is a fatal error
	userbackends.DB, code, err = localdb.NewClientFromCredentials(Config.LocalDB, body.Username, body.Password)
	if err != nil {
		c.JSON(code, gin.H{"auth": false, "error": err.Error()})
		return
	}

	// successful binding at this point
	// create new session
	session := sessions.Default(c)
	// create random uuid to map user to backends
	uuid, _ := uuid.NewV4()
	// set uuid mapping in session
	session.Set("SessionUUID", uuid.String())
	// set uuid mapping in LDAPSessions
	UserSessions[uuid.String()] = &userbackends
	// save the session
	err = session.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"auth": false, "error": err.Error()})
	} else {
		// return successful auth
		c.JSON(http.StatusOK, gin.H{"auth": true})
	}
}

func DELETE_Ticket(c *gin.Context) {
	// get session uuid from session cookie
	session := sessions.Default(c)
	SessionUUID := session.Get("SessionUUID")
	if SessionUUID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"auth": false})
		return
	}
	uuid := SessionUUID.(string)

	// delete uuid entry from user sessions
	delete(UserSessions, uuid)                    // deletes uuid mapping
	session.Options(sessions.Options{MaxAge: -1}) // set max age to -1 so session cookie is deleted
	err := session.Save()                         // if save somehow fails, it should be ok since the uuid mapping is already deleted
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"auth": false})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"auth": false})
	}
}

func GET_Pool(c *gin.Context) {
	poolid, ok := c.Params.Get("poolid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	pool, code, err := GetPool(backends, poolid)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"pool": pool})
	}
}

func POST_Pool(c *gin.Context) {
	poolid, ok := c.Params.Get("poolid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	// read request body
	pool := common.Pool{}
	err = c.ShouldBindJSON(&pool)
	if err != nil { // failed binding, usually missing user field
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, code, _ = GetPool(backends, poolid) // test pool existence

	if code == http.StatusNotFound { // pool does not already exist, create new pool
		code, err = NewPool(backends, poolid, pool)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(http.StatusOK)
		}
	} else {
		code, err = ModPool(backends, poolid, pool)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(http.StatusOK)
		}
	}
}

func DELETE_Pool(c *gin.Context) {
	poolid, ok := c.Params.Get("poolid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	code, err = DelPool(backends, poolid)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.Status(http.StatusOK)
	}
}

func GET_Group(c *gin.Context) {
	groupname_str, ok := c.Params.Get("groupname")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	groupname, err := common.ParseGroupname(groupname_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	group, code, err := GetGroup(backends, groupname)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"group": group})
	}
}

func POST_Group(c *gin.Context) {
	groupname_str, ok := c.Params.Get("groupname")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	// read request body
	group := common.Group{}
	err = c.ShouldBindJSON(&group)
	if err != nil { // failed binding, usually missing user field
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, code, _ = GetGroup(backends, groupname) // test group existence

	if code == http.StatusNotFound { // group does not already exist, create new group
		code, err = NewGroup(backends, groupname, group)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(http.StatusOK)
		}
	} else {
		code, err = ModGroup(backends, groupname, group)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(http.StatusOK)
		}
	}
}

func DELETE_Group(c *gin.Context) {
	groupname_str, ok := c.Params.Get("groupname")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	code, err = DelGroup(backends, groupname)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.Status(http.StatusOK)
	}
}

func POST_Pool_Group(c *gin.Context) {
	poolid, ok := c.Params.Get("poolid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	groupname_str, ok := c.Params.Get("groupname")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	code, err = AddGroupToPool(backends, groupname, poolid)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.Status(http.StatusOK)
	}
}

func DELETE_Pool_Group(c *gin.Context) {
	poolid, ok := c.Params.Get("poolid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	groupname_str, ok := c.Params.Get("groupname")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	code, err = DelGroupFromPool(backends, groupname, poolid)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.Status(http.StatusOK)
	}
}

func GET_User(c *gin.Context) {
	username_str, ok := c.Params.Get("username")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	username, err := common.ParseUsername(username_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	user, code, err := GetUser(backends, username)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

func POST_User(c *gin.Context) {
	username_str, ok := c.Params.Get("username")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	username, err := paas.ParseUsername(username_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	// read request body
	user := common.User{}
	err = c.ShouldBindJSON(&user)
	if err != nil { // failed binding, usually missing user field
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, code, _ = GetUser(backends, username) // test user existence

	if code == http.StatusNotFound { // user does not already exist, create new user
		code, err = NewUser(backends, username, user)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(http.StatusOK)
		}
	} else {
		code, err = ModUser(backends, username, user)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(http.StatusOK)
		}
	}
}

func DELETE_User(c *gin.Context) {
	username_str, ok := c.Params.Get("username")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	username, err := paas.ParseUsername(username_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	code, err = DelUser(backends, username)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.Status(http.StatusOK)
	}
}

func POST_Group_User(c *gin.Context) {
	groupname_str, ok := c.Params.Get("groupname")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	username_str, ok := c.Params.Get("username")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter username")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	username, err := paas.ParseUsername(username_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	code, err = AddUserToGroup(backends, username, groupname)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.Status(http.StatusOK)
	}
}

func DELETE_Group_User(c *gin.Context) {
	groupname_str, ok := c.Params.Get("groupname")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	username_str, ok := c.Params.Get("username")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("missing required path parameter username")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	username, err := paas.ParseUsername(username_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromContext(c)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
		return
	}

	code, err = DelUserFromGroup(backends, username, groupname)
	if err != nil {
		c.JSON(code, gin.H{"error": err.Error()})
	} else {
		c.Status(http.StatusOK)
	}
}
