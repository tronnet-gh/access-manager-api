package app

import (
	"crypto/rand"
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
	router := SetupAPI(&Config)

	// make global session map
	UserSessions = make(map[string]*Backends)

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

		// bind proxmox backend
		newPVEClient, code, err := pve.NewClientFromCredentials(Config.PVE, body.Username, body.Password)
		if err != nil { // pve client failed to bind
			c.JSON(code, gin.H{"auth": false, "error": err.Error()})
			return
		}

		// bind ldap backend
		newLDAPClient, code, err := ldap.NewClientFromCredentials(Config.LDAP, body.Username, body.Password)
		if err != nil { // ldap client failed to bind
			c.JSON(code, gin.H{"auth": false, "error": err.Error()})
			return
		}
		//err = newLDAPClient.BindUser(body.Username, body.Password)
		//if err != nil { // failed to authenticate, return error
		//	c.JSON(http.StatusBadRequest, gin.H{"auth": false, "error": err.Error()})
		//	return
		//}
		// todo allow ldap backed to fail if user is not using an ldap backend

		// successful binding at this point
		// create new session
		session := sessions.Default(c)
		// create (hopefully) safe uuid to map to ldap session
		uuid, _ := uuid.NewV4()
		// set uuid mapping in session
		session.Set("SessionUUID", uuid.String())
		// set uuid mapping in LDAPSessions
		UserSessions[uuid.String()] = &Backends{pve: newPVEClient, ldap: newLDAPClient}
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

	router.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{"version": Version})
	})

	router.POST("/pool/:poolid", func(c *gin.Context) {
		poolid, _ := c.Params.Get("poolid")

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

	router.DELETE("/pool/:poolid", func(c *gin.Context) {
		poolid, _ := c.Params.Get("poolid")

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

	log.Printf("Starting User Manager API on port %s\n", strconv.Itoa(Config.ListenPort))

	err = router.Run("0.0.0.0:" + strconv.Itoa(Config.ListenPort))
	if err != nil {
		log.Fatalf("Error starting router: %s", err.Error())
	}
}

func SetupAPI(config *common.Config) *gin.Engine {
	secretKey := make([]byte, 256)
	n, err := rand.Read(secretKey)
	if err != nil {
		log.Fatalf("Error when generating session secret key: %s\n", err.Error())
	}
	log.Printf("Generated session secret key of length %d\n", n)

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	store := cookie.NewStore(secretKey)
	store.Options(sessions.Options{
		Path:     config.SessionCookie.Path,
		HttpOnly: config.SessionCookie.HttpOnly,
		Secure:   config.SessionCookie.Secure,
		MaxAge:   config.SessionCookie.MaxAge,
	})
	router.Use(sessions.Sessions(config.SessionCookieName, store))

	log.Printf("Started API router and cookie store (Name: %s Params: %+v)\n", config.SessionCookieName, config.SessionCookie)

	return router
}
