package app

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"strconv"

	common "user-manager-api/app/common"
	ldap "user-manager-api/app/ldap"
	"user-manager-api/app/pve"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	uuid "github.com/nu7hatch/gouuid"
)

var Version = "0.0.1"
var Config common.Config
var UserSessions map[string]*Backends

func Run(configPath *string) {
	// load config values
	Config, err := common.GetConfig(*configPath)
	if err != nil {
		log.Fatalf("Error when reading config file: %s\n", err)
	}
	log.Printf("Read in config from %s\n", *configPath)

	// setup router
	router := SetupAPISessionStore(&Config)
	gin.SetMode(gin.ReleaseMode)

	// make global session map
	UserSessions = make(map[string]*Backends)

	router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": Version})
	})

	router.POST("/ticket", func(c *gin.Context) {
		body := common.Login{}
		if err := c.ShouldBind(&body); err != nil { // bad request from binding
			c.JSON(http.StatusBadRequest, gin.H{"auth": false, "error": err.Error()})
			return
		}

		// attempt to parse username
		body.Username, err = common.ParseUsername(body.UsernameRaw)
		if err != nil { // username format incorrect
			c.JSON(http.StatusBadRequest, gin.H{"auth": false, "error": err.Error()})
			return
		}
		handler := Config.Realms[body.Username.Realm].Handler

		// always bind proxmox backend
		PVEClient, code, err := pve.NewClientFromCredentials(Config.PVE, body.Username, body.Password)
		if err != nil { // pve client failed to bind
			c.JSON(code, gin.H{"auth": false, "error": err.Error()})
			return
		}

		// bind ldap backend if backend is ldap
		var LDAPClient *ldap.LDAPClient
		if handler == "ldap" {
			LDAPClient, code, err = ldap.NewClientFromCredentials(Config.LDAP, body.Username, body.Password)
			if err != nil { // ldap client failed to bind
				c.JSON(code, gin.H{"auth": false, "error": err.Error()})
				return
			}
		} //ldap client will be nil if it is unused!!

		// successful binding at this point
		// create new session
		session := sessions.Default(c)
		// create random uuid to map user to backends
		uuid, _ := uuid.NewV4()
		// set uuid mapping in session
		session.Set("SessionUUID", uuid.String())
		// set uuid mapping in LDAPSessions
		UserSessions[uuid.String()] = &Backends{handler: handler, pve: PVEClient, ldap: LDAPClient}
		// save the session
		session.Save()
		// return successful auth
		c.JSON(http.StatusOK, gin.H{"auth": true})
	})

	router.DELETE("/ticket", func(c *gin.Context) {
		session := sessions.Default(c)
		SessionUUID := session.Get("SessionUUID")
		if SessionUUID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"auth": false})
			return
		}
		uuid := SessionUUID.(string)
		delete(UserSessions, uuid)
		session.Options(sessions.Options{MaxAge: -1}) // set max age to -1 so it is deleted
		session.Save()
		c.JSON(http.StatusUnauthorized, gin.H{"auth": false})
	})

	router.POST("/pools/:poolid", func(c *gin.Context) {
		poolid, ok := c.Params.Get("poolid")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("Missing required path parameter poolid")})
		}

		backends, code, err := GetBackendsFromContext(c)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		}

		code, err = NewPool(backends, poolid)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(200)
		}
	})

	router.DELETE("/pools/:poolid", func(c *gin.Context) {
		poolid, ok := c.Params.Get("poolid")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("Missing required path parameter poolid")})
		}

		backends, code, err := GetBackendsFromContext(c)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		}

		code, err = DelPool(backends, poolid)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(200)
		}
	})

	router.POST("/groups/:groupid", func(c *gin.Context) {
		groupid, ok := c.Params.Get("groupid")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("Missing required path parameter groupid")})
		}
		groupname, err := common.ParseGroupname(groupid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}

		backends, code, err := GetBackendsFromContext(c)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		}

		code, err = NewGroup(backends, groupname)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(200)
		}
	})

	router.DELETE("/groups/:groupid", func(c *gin.Context) {
		groupid, ok := c.Params.Get("groupid")
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Errorf("Missing required path parameter groupid")})
		}
		groupname, err := common.ParseGroupname(groupid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err})
		}

		backends, code, err := GetBackendsFromContext(c)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		}

		code, err = DelGroup(backends, groupname)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(200)
		}
	})

	log.Printf("Starting User Manager API on port %s\n", strconv.Itoa(Config.ListenPort))

	err = router.Run("0.0.0.0:" + strconv.Itoa(Config.ListenPort))
	if err != nil {
		log.Fatalf("Error starting router: %s", err.Error())
	}
}

func SetupAPISessionStore(config *common.Config) *gin.Engine {
	secretKey := make([]byte, 256)
	n, _ := rand.Read(secretKey) // rand Read never returns an error, always crashes on error
	log.Printf("Generated session secret key of length %d\n", n)

	router := gin.Default()
	store := cookie.NewStore(secretKey)
	store.Options(sessions.Options{
		Path:     config.SessionCookie.Path,
		HttpOnly: config.SessionCookie.HttpOnly,
		Secure:   config.SessionCookie.Secure,
		MaxAge:   config.SessionCookie.MaxAge,
	})
	router.Use(sessions.Sessions(config.SessionCookieName, store))

	log.Printf("Started cookie store (Name: %s Params: %+v)\n", config.SessionCookieName, config.SessionCookie)

	return router
}
