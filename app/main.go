package app

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	common "access-manager-api/app/common"
	"access-manager-api/app/pve"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

type UserSession struct {
	PVE   common.Backend
	Realm struct {
		Name    string
		Handler common.Backend
	}
	DB common.Backend
}

var Version = "1.0.0"
var Config common.Config
var RootSession *UserSession
var UserSessions map[string]*UserSession
var Realms map[string]common.Realm

func Run() {
	configPath := flag.String("config", "config.json", "path to config.json file")
	flag.Parse()

	// load config values
	Config = common.GetConfig(*configPath)
	// already exits if failed
	log.Printf("Read in config from %s\n", *configPath)

	// setup gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// setup api auth cookies
	SetupAPISessionStore(router, &Config)

	// setup root api token
	client, code, err := pve.NewClientFromAPIToken(Config.PVE, Config.PVE.Token)
	if err != nil {
		log.Fatalf("error initializing pve root client: %d %s", code, err)
	}
	RootSession = &UserSession{
		PVE: client,
	}

	// get realms from proxmox
	Realms = make(map[string]common.Realm)
	Realms = RootSession.PVE.(pve.ProxmoxClient).GetRealms()

	// make global session map
	UserSessions = make(map[string]*UserSession)

	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": Version})
	})

	router.POST("/ticket", POST_Ticket)
	router.DELETE("/ticket", DELETE_Ticket)
	router.GET("/pools/:poolid", GET_Pool)
	router.POST("/pools/:poolid", POST_Pool)
	router.DELETE("/pools/:poolid", DELETE_Pool)
	router.GET("/groups/:groupname", GET_Group)
	router.POST("/groups/:groupname", POST_Group)
	router.DELETE("/groups/:groupname", DELETE_Group)
	router.POST("/pools/:poolid/groups/:groupname", POST_Pool_Group)
	router.DELETE("/pools/:poolid/groups/:groupname")
	router.GET("/users/:username", GET_User)
	router.POST("/users/:username", POST_User)
	router.DELETE("/users/:username", DELETE_User)
	router.POST("/groups/:groupname/users/:username", POST_Group_User)
	router.DELETE("/groups/:groupname/users/:username", DELETE_Group_User)

	log.Printf("Starting Access Manager API on port %s\n", strconv.Itoa(Config.ListenPort))

	err = router.Run("0.0.0.0:" + strconv.Itoa(Config.ListenPort))
	if err != nil {
		log.Fatalf("Error starting router: %s", err.Error())
	}
}

func SetupAPISessionStore(router *gin.Engine, config *common.Config) {
	authKey := make([]byte, 32)
	encrKey := make([]byte, 32)
	n, _ := rand.Read(authKey) // rand Read never returns an error, always crashes on error
	log.Printf("Generated cookie session authentication key of length %d\n", n)
	n, _ = rand.Read(encrKey) // rand Read never returns an error, always crashes on error
	log.Printf("Generated cookie session encryption key of length %d\n", n)

	store := cookie.NewStore(authKey, encrKey)
	store.Options(sessions.Options{
		Path:     config.SessionCookie.Path,
		HttpOnly: config.SessionCookie.HttpOnly,
		Secure:   config.SessionCookie.Secure,
		MaxAge:   config.SessionCookie.MaxAge,
	})
	router.Use(sessions.Sessions(config.SessionCookieName, store))

	log.Printf("Started cookie store (Name: %s Params: %+v)\n", config.SessionCookieName, config.SessionCookie)
}

func GetUserSessionFromContext(c *gin.Context) (*UserSession, int, error) {
	session := sessions.Default(c)
	SessionUUID := session.Get("SessionUUID")
	if SessionUUID == nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("no auth session found")
	}
	uuid := SessionUUID.(string)
	usersession := UserSessions[uuid]
	return usersession, http.StatusOK, nil
}
