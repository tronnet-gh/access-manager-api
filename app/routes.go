package app

import (
	common "access-manager-api/app/common"
	ldap "access-manager-api/app/ldap"
	"access-manager-api/app/localdb"
	pve "access-manager-api/app/pve"
	paas "access-manager-api/proxmoxaas-common-lib"
	"encoding/json"
	"fmt"
	"net/http"
	"uuid"

	"github.com/gorilla/sessions"
)

func POST_Ticket(w http.ResponseWriter, r *http.Request) {
	body := common.Login{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil { // bad request from decoding
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"auth": false, "error": err.Error()})
		return
	}
	if body.UsernameRaw == "" || body.Password == "" { // username and password are required
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"auth": false, "error": "missing required field username or password"})
		return
	}

	// attempt to parse username
	body.Username, err = paas.ParseUsername(body.UsernameRaw)
	if err != nil { // username format incorrect
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"auth": false, "error": err.Error()})
		return
	}
	handler := Realms[body.Username.Realm].Type

	userbackends := UserSession{}

	// always bind proxmox backend
	PVEClient, code, err := pve.NewClientFromCredentials(Config.PVE, body.Username, body.Password)
	if err != nil { // pve client failed to bind
		paas.HTTPWriteJson(w, code, map[string]any{"auth": false, "error": err.Error()})
		return
	}
	userbackends.PVE = PVEClient

	// bind backend by type
	switch handler {
	case "pve":
	case "ldap":
		config := Realms[body.Username.Realm].Config.(common.LDAPConfig)
		config.RealmName = body.Username.Realm
		LDAPClient, code, err := ldap.NewClientFromCredentials(config, body.Username, body.Password)
		if err != nil { // ldap client failed to bind
			paas.HTTPWriteJson(w, code, map[string]any{"auth": false, "error": err.Error()})
			return
		}
		userbackends.Realm.Name = body.Username.Realm
		userbackends.Realm.Handler = LDAPClient
	default:
		paas.HTTPWriteJson(w, code, map[string]any{"auth": false, "error": fmt.Errorf("user realm %s is not supported", body.Username.Realm)})
		return
	}

	// failing to load lcoaldb is a fatal error
	userbackends.DB, code, err = localdb.NewClientFromCredentials(Config.LocalDB, body.Username, body.Password)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"auth": false, "error": err.Error()})
		return
	}

	// successful binding at this point
	// create new session
	session := GetCookieSession(r)
	// create random uuid to map user to backends
	uuid := uuid.NewV4()
	// set uuid mapping in session
	session.Values["SessionUUID"] = uuid.String()
	// set uuid mapping in LDAPSessions
	UserSessions[uuid.String()] = &userbackends
	// save the session
	err = session.Save(r, w)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusInternalServerError, map[string]any{"auth": false, "error": err.Error()})
	} else {
		// return successful auth
		paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"auth": true})
	}
}

func DELETE_Ticket(w http.ResponseWriter, r *http.Request) {
	// get session uuid from session cookie
	session := GetCookieSession(r)
	SessionUUID := session.Values["SessionUUID"]
	if SessionUUID == nil {
		paas.HTTPWriteJson(w, http.StatusUnauthorized, map[string]any{"auth": false})
		return
	}
	uuid := SessionUUID.(string)

	// delete uuid entry from user sessions
	delete(UserSessions, uuid)                      // deletes uuid mapping
	session.Options = &sessions.Options{MaxAge: -1} // set max age to -1 so session cookie is deleted
	err := session.Save(r, w)                       // if save somehow fails, it should be ok since the uuid mapping is already deleted
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusInternalServerError, map[string]any{"auth": false})
	} else {
		paas.HTTPWriteJson(w, http.StatusUnauthorized, map[string]any{"auth": false})
	}
}

func GET_Pools(w http.ResponseWriter, r *http.Request) {
	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	poolnames, code, err := GetAllPoolnames(backends)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"poolnames": poolnames})
	}
}

func GET_Pool(w http.ResponseWriter, r *http.Request) {
	poolid := r.PathValue("poolid")
	if poolid == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	pool, code, err := GetPool(backends, poolid)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"pool": pool})
	}
}

func POST_Pool(w http.ResponseWriter, r *http.Request) {
	poolid := r.PathValue("poolid")
	if poolid == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	// read request body
	pool := common.Pool{}
	err = json.NewDecoder(r.Body).Decode(&pool)
	if err != nil { // failed decoding, usually malformed json
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	_, code, _ = GetPool(backends, poolid) // test pool existence

	if code == http.StatusNotFound { // pool does not already exist, create new pool
		code, err = NewPool(backends, poolid, pool)
		if err != nil {
			paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		code, err = ModPool(backends, poolid, pool)
		if err != nil {
			paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func DELETE_Pool(w http.ResponseWriter, r *http.Request) {
	poolid := r.PathValue("poolid")
	if poolid == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	code, err = DelPool(backends, poolid)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func GET_Groups(w http.ResponseWriter, r *http.Request) {
	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	groupnames, code, err := GetAllGroupnames(backends)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"groupnames": groupnames})
	}
}

func GET_Group(w http.ResponseWriter, r *http.Request) {
	groupname_str := r.PathValue("groupname")
	if groupname_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	groupname, err := common.ParseGroupname(groupname_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	group, code, err := GetGroup(backends, groupname)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"group": group})
	}
}

func POST_Group(w http.ResponseWriter, r *http.Request) {
	groupname_str := r.PathValue("groupname")
	if groupname_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	// read request body
	group := common.Group{}
	err = json.NewDecoder(r.Body).Decode(&group)
	if err != nil { // failed decoding, usually malformed json
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	_, code, _ = GetGroup(backends, groupname) // test group existence

	if code == http.StatusNotFound { // group does not already exist, create new group
		code, err = NewGroup(backends, groupname, group)
		if err != nil {
			paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		code, err = ModGroup(backends, groupname, group)
		if err != nil {
			paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func DELETE_Group(w http.ResponseWriter, r *http.Request) {
	groupname_str := r.PathValue("groupname")
	if groupname_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	code, err = DelGroup(backends, groupname)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func POST_Pool_Group(w http.ResponseWriter, r *http.Request) {
	poolid := r.PathValue("poolid")
	if poolid == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	groupname_str := r.PathValue("groupname")
	if groupname_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	code, err = AddGroupToPool(backends, groupname, poolid)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func DELETE_Pool_Group(w http.ResponseWriter, r *http.Request) {
	poolid := r.PathValue("poolid")
	if poolid == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	groupname_str := r.PathValue("groupname")
	if groupname_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	code, err = DelGroupFromPool(backends, groupname, poolid)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func GET_Users(w http.ResponseWriter, r *http.Request) {
	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	usernames, code, err := GetAllUsernames(backends)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"usernames": usernames})
	}
}

func GET_User(w http.ResponseWriter, r *http.Request) {
	username_str := r.PathValue("username")
	if username_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter poolid")})
		return
	}

	username, err := common.ParseUsername(username_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	user, code, err := GetUser(backends, username)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"user": user})
	}
}

func POST_User(w http.ResponseWriter, r *http.Request) {
	username_str := r.PathValue("username")
	if username_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	username, err := paas.ParseUsername(username_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	// read request body
	user := common.User{}
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil { // failed decoding, usually malformed json
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	_, code, _ = GetUser(backends, username) // test user existence

	if code == http.StatusNotFound { // user does not already exist, create new user
		code, err = NewUser(backends, username, user)
		if err != nil {
			paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	} else {
		code, err = ModUser(backends, username, user)
		if err != nil {
			paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func DELETE_User(w http.ResponseWriter, r *http.Request) {
	username_str := r.PathValue("username")
	if username_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	username, err := paas.ParseUsername(username_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	code, err = DelUser(backends, username)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func POST_Group_User(w http.ResponseWriter, r *http.Request) {
	groupname_str := r.PathValue("groupname")
	if groupname_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	username_str := r.PathValue("username")
	if username_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter username")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	username, err := paas.ParseUsername(username_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	code, err = AddUserToGroup(backends, username, groupname)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func DELETE_Group_User(w http.ResponseWriter, r *http.Request) {
	groupname_str := r.PathValue("groupname")
	if groupname_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter groupname")})
		return
	}

	username_str := r.PathValue("username")
	if username_str == "" {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Errorf("missing required path parameter username")})
		return
	}

	groupname, err := paas.ParseGroupname(groupname_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	username, err := paas.ParseUsername(username_str)
	if err != nil {
		paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": err})
		return
	}

	backends, code, err := GetUserSessionFromRequest(r)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
		return
	}

	code, err = DelUserFromGroup(backends, username, groupname)
	if err != nil {
		paas.HTTPWriteJson(w, code, map[string]any{"error": err.Error()})
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
