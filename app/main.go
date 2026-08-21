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

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/luthermonson/go-proxmox"
)

type Realm struct {
	Type   string
	Config any
}

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
var UserSessions map[string]*UserSession
var Realms map[string]Realm

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

	// get realms from proxmox
	Realms = make(map[string]Realm)
	Realms = GetRealmsFromPVE(&Config)

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

	err := router.Run("0.0.0.0:" + strconv.Itoa(Config.ListenPort))
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
