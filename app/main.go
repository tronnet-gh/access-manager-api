package app

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	common "access-manager-api/app/common"
	ldap "access-manager-api/app/ldap"
	localdb "access-manager-api/app/localdb"
	pve "access-manager-api/app/pve"
	paas "proxmoxaas-common-lib"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/luthermonson/go-proxmox"
	uuid "github.com/nu7hatch/gouuid"
)

var Version = "1.0.0"
var Config common.Config
var UserSessions map[string]*UserSession
var Realms map[string]Realm

func Run() {
	configPath := flag.String("config", "config.json", "path to config.json file")
	localDBPath := flag.String("localdb", "localdb.json", "path to localdb.json file")
	flag.Parse()

	// load config values
	var err error
	Config = common.GetConfig(*configPath)
	// already exits if failed
	log.Printf("Read in config from %s\n", *configPath)

	// load localdb
	db, err := localdb.LoadDB(*localDBPath)
	if err != nil {
		log.Fatalf("Error when reading localdb file: %s\n", err)
	}
	log.Printf("Read in localdb from %s\n", *localDBPath)

	// setup router
	gin.SetMode(gin.ReleaseMode)
	router := SetupAPISessionStore(&Config)

	// get realms from proxmox
	Realms = make(map[string]Realm)
	Realms = GetRealmsFromPVE(&Config)

	// make global session map
	UserSessions = make(map[string]*UserSession)

	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": Version})
	})

	router.POST("/ticket", func(c *gin.Context) {
		body := common.Login{}
		err := c.ShouldBind(&body)
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

		userbackends.DB = &db

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
			c.JSON(http.StatusInternalServerError, gin.H{"auth": false})
		} else {
			// return successful auth
			c.JSON(http.StatusOK, gin.H{"auth": true})
		}
	})

	router.DELETE("/ticket", func(c *gin.Context) {
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
	})

	router.GET("/pools/:poolid", func(c *gin.Context) {
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
	})

	router.POST("/pools/:poolid", func(c *gin.Context) {
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

		code, err = NewPool(backends, poolid)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(http.StatusOK)
		}
	})

	router.DELETE("/pools/:poolid", func(c *gin.Context) {
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
	})

	router.GET("/groups/:groupname", func(c *gin.Context) {
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
	})

	router.POST("/groups/:groupname", func(c *gin.Context) {
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

		code, err = NewGroup(backends, groupname)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(http.StatusOK)
		}
	})

	router.DELETE("/groups/:groupname", func(c *gin.Context) {
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
	})

	router.POST("/pools/:poolid/groups/:groupname", func(c *gin.Context) {
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
	})

	router.DELETE("/pools/:poolid/groups/:groupname", func(c *gin.Context) {
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
	})

	router.GET("/users/:username", func(c *gin.Context) {
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
	})

	router.POST("/users/:username", func(c *gin.Context) {
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

		form := common.UserFormRequired{}
		err = c.ShouldBind(&form)
		if err != nil { // failed binding, usually missing form field
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user := common.User{}
		user.CN = form.CN
		user.SN = form.SN
		user.Mail = form.Mail
		user.Password = form.Password

		code, err = NewUser(backends, username, user)
		if err != nil {
			c.JSON(code, gin.H{"error": err.Error()})
		} else {
			c.Status(http.StatusOK)
		}
	})

	router.DELETE("/users/:username", func(c *gin.Context) {
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
	})

	router.POST("/groups/:groupname/users/:username", func(c *gin.Context) {
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
	})

	router.DELETE("/groups/:groupname/users/:username", func(c *gin.Context) {
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
	})

	log.Printf("Starting Access Manager API on port %s\n", strconv.Itoa(Config.ListenPort))

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

func GetRealmsFromPVE(config *common.Config) map[string]Realm {
	realms := map[string]Realm{}

	HTTPClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{},
		},
	}
	token := fmt.Sprintf(`%s@%s!%s`, config.PVE.Token.User, config.PVE.Token.Realm, config.PVE.Token.ID)
	client := proxmox.NewClient(config.PVE.URL,
		proxmox.WithHTTPClient(&HTTPClient),
		proxmox.WithAPIToken(token, config.PVE.Token.UUID),
	)

	pverealms, err := client.Domains(context.Background())
	if err != nil {
		// failure to get realms is a fatal error
		log.Fatalf("Error getting authentication realms: %s", err.Error())
	}

	// add required pve realm handler, removing the pve api token
	pveconfig := common.PVEConfig{
		URL:            config.PVE.URL,
		PAASClientRole: config.PVE.PAASClientRole,
	}
	realms["pve"] = Realm{
		Type:   "pve",
		Config: pveconfig,
	}
	log.Printf("Configured default authentication realm pve")

	// iterate through handlers and add to realms
	for _, r := range pverealms {
		realm, err := client.Domain(context.Background(), r.Realm)
		if err != nil {
			log.Printf("Error getting authentication realm %s: %s", r.Realm, err.Error())
		}

		if realm.Type == "ldap" {
			ldapconfig := common.LDAPConfig{
				BaseDN:   realm.BaseDN,
				Hostname: realm.Server1,
				TLS:      realm.Mode == "ldaps",
				StartTLS: realm.Mode == "ldap+starttls",
				Verify:   bool(realm.Verify),
			}
			realms[realm.Realm] = Realm{
				Type:   realm.Type,
				Config: ldapconfig,
			}
			log.Printf("Configured external authentication realm %s", realm.Realm)
		} else {
			continue
		}
	}

	return realms
}
