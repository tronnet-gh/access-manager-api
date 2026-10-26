package app

import (
	common "access-manager-api/app/common"
	"access-manager-api/app/pve"
	paas "access-manager-api/proxmoxaas-common-lib"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/sessions"
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
var Store *sessions.CookieStore

func Run() {
	configPath := flag.String("config", "config.json", "path to config.json file")
	flag.Parse()

	// load config values
	Config = common.GetConfig(*configPath)
	// already exits if failed
	log.Printf("[INFO] Read in config from %s\n", *configPath)

	// setup http router
	router := http.NewServeMux()

	// setup api auth cookies
	SetupAPISessionStore(&Config)

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

	router.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"version": Version})
	})

	router.HandleFunc("POST /ticket", POST_Ticket)
	router.HandleFunc("DELETE /ticket", DELETE_Ticket)
	router.HandleFunc("GET /pools/{poolid}", GET_Pool)
	router.HandleFunc("POST /pools/{poolid}", POST_Pool)
	router.HandleFunc("DELETE /pools/{poolid}", DELETE_Pool)
	router.HandleFunc("GET /groups/{groupname}", GET_Group)
	router.HandleFunc("POST /groups/{groupname}", POST_Group)
	router.HandleFunc("DELETE /groups/{groupname}", DELETE_Group)
	router.HandleFunc("POST /pools/{poolid}/groups/{groupname}", POST_Pool_Group)
	router.HandleFunc("DELETE /pools/{poolid}/groups/{groupname}", DELETE_Pool_Group)
	router.HandleFunc("GET /users/{username}", GET_User)
	router.HandleFunc("POST /users/{username}", POST_User)
	router.HandleFunc("DELETE /users/{username}", DELETE_User)
	router.HandleFunc("POST /groups/{groupname}/users/{username}", POST_Group_User)
	router.HandleFunc("DELETE /groups/{groupname}/users/{username}", DELETE_Group_User)

	server := paas.LoggingMiddleware(router)

	log.Printf("[INFO] starting API listening on 0.0.0.0:%d", Config.ListenPort)
	log.Fatalf("[ERR ] Error starting router: %s", http.ListenAndServe("0.0.0.0:"+strconv.Itoa(Config.ListenPort), server))
}

func SetupAPISessionStore(config *common.Config) {
	authKey := make([]byte, 32)
	encrKey := make([]byte, 32)
	n, _ := rand.Read(authKey) // rand Read never returns an error, always crashes on error
	log.Printf("Generated cookie session authentication key of length %d\n", n)
	n, _ = rand.Read(encrKey) // rand Read never returns an error, always crashes on error
	log.Printf("Generated cookie session encryption key of length %d\n", n)

	Store = sessions.NewCookieStore(authKey, encrKey)
	Store.Options = &sessions.Options{
		Path:     config.SessionCookie.Path,
		HttpOnly: config.SessionCookie.HttpOnly,
		Secure:   config.SessionCookie.Secure,
		MaxAge:   config.SessionCookie.MaxAge,
	}

	log.Printf("Started cookie store (Name: %s Params: %+v)\n", config.SessionCookieName, config.SessionCookie)
}

// get the cookie session from the request, returns a new session if the cookie is missing or invalid
func GetCookieSession(r *http.Request) *sessions.Session {
	session, err := Store.Get(r, Config.SessionCookieName)
	if err != nil {
		log.Printf("Error getting session cookie: %s\n", err.Error())
	}
	return session
}

func GetUserSessionFromRequest(r *http.Request) (*UserSession, int, error) {
	session := GetCookieSession(r)
	SessionUUID := session.Values["SessionUUID"]
	if SessionUUID == nil {
		return nil, http.StatusUnauthorized, fmt.Errorf("no auth session found")
	}
	uuid := SessionUUID.(string)
	usersession := UserSessions[uuid]
	return usersession, http.StatusOK, nil
}
