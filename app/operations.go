package app

import (
	common "access-manager-api/app/common"
	"access-manager-api/app/ldap"
	"fmt"
	"net/http"
)

func NewPool(backends *UserSession, poolname string) (int, error) {
	// only pve backend handles pools
	return backends.PVE.NewPool(poolname)
}

// get pool recursive resolving groups
func GetPool(backends *UserSession, poolname string) (common.Pool, int, error) {
	pool := common.Pool{}
	// get pool from PVE
	pvepool, members, code, err := backends.PVE.GetPool(poolname)
	if err != nil {
		return pool, code, err
	}
	// get pool from DB
	dbpool, _, code, err := backends.DB.GetPool(poolname)
	if err != nil {
		return pool, code, err
	}
	// assign pool id from PVE, assign everything else from DB
	pool.PoolID = pvepool.PoolID
	pool.Resources = dbpool.Resources
	pool.AllowedNodes = dbpool.AllowedNodes
	pool.AllowedVMIDRange = dbpool.AllowedVMIDRange
	pool.AllowedBackups = dbpool.AllowedBackups
	pool.Templates = dbpool.Templates

	// pool members are groups
	for _, groupid := range members {
		groupname, err := common.ParseGroupname(groupid)
		if err != nil {
			return pool, code, fmt.Errorf("pool %s had member %s which is not a valid groupname", poolname, groupid)
		}
		group, code, err := GetGroup(backends, groupname)
		if err != nil {
			return pool, code, err
		}
		group.Role = Config.PVE.PAASClientRole
		pool.Groups = append(pool.Groups, group)
	}

	return pool, http.StatusOK, nil
}
func DelPool(backends *UserSession, poolname string) (int, error) {
	// only pve backend handles pools
	return backends.PVE.DelPool(poolname)
}

func NewGroup(backends *UserSession, groupname common.Groupname) (int, error) {
	if groupname.Realm == "pve" {
		return backends.PVE.NewGroup(groupname)
	} else if groupname.Realm == backends.Realm.Name {
		realm_handler := backends.Realm.Handler.(common.Backend)
		code, err := realm_handler.NewGroup(groupname)
		if err != nil {
			return code, err
		}
		return backends.PVE.SyncRealms()
	} else {
		return http.StatusUnauthorized, fmt.Errorf("user is not in the same realm as requested group")
	}
}

func GetGroup(backends *UserSession, groupname common.Groupname) (common.Group, int, error) {
	// resolve group from relevant backend
	if groupname.Realm == "pve" {
		group, members, code, err := backends.PVE.GetGroup(groupname)
		if err != nil {
			return group, code, err
		}
		// group members are users
		for _, userid := range members {
			username, err := common.ParseUsername(userid)
			if err != nil {
				return group, http.StatusInternalServerError, fmt.Errorf("group %s had member %s which is not a valid username", groupname.ToString(), userid)
			}
			// fetch and append user
			user, code, err := GetUser(backends, username)
			if err != nil {
				return group, code, err
			}
			group.Users = append(group.Users, user)
		}

		return group, http.StatusOK, nil
	} else if groupname.Realm == backends.Realm.Name {
		group, members, code, err := backends.Realm.Handler.(common.Backend).GetGroup(groupname)
		if err != nil {
			return common.Group{}, code, err
		}
		// group mambers are users
		for _, userdn := range members {
			// member list is of ldap user DN instead of pve userid
			ldapuid, err := ldap.ExtractUIDFromUserDN(userdn)
			if err != nil {
				return group, http.StatusInternalServerError, fmt.Errorf("group %s had member %s which is not a valid user DN", groupname.ToString(), userdn)
			}
			// generate username from user DN (slightly inefficient)
			userid := fmt.Sprintf("%s@%s", ldapuid, backends.Realm.Name)
			username, err := common.ParseUsername(userid)
			if err != nil {
				return group, http.StatusInternalServerError, fmt.Errorf("group %s had member %s which is not a valid username", groupname.ToString(), userid)
			}
			// fetch and append user
			user, code, err := GetUser(backends, username)
			if err != nil {
				return group, code, err
			}
			group.Users = append(group.Users, user)
		}

		return group, http.StatusOK, nil
	} else {
		return common.Group{}, http.StatusUnauthorized, fmt.Errorf("user is not in the same realm as requested group")
	}
}

func DelGroup(backends *UserSession, groupname common.Groupname) (int, error) {
	if groupname.Realm == "pve" {
		return backends.PVE.DelGroup(groupname)
	} else if groupname.Realm == backends.Realm.Name {
		realm_handler := backends.Realm.Handler.(common.Backend)
		code, err := realm_handler.DelGroup(groupname)
		if err != nil {
			return code, err
		}
		return backends.PVE.SyncRealms()
	} else {
		return http.StatusUnauthorized, fmt.Errorf("user is not in the same realm as requested group")
	}
}

func AddGroupToPool(backends *UserSession, groupname common.Groupname, poolname string) (int, error) {
	// only pve backend handles pool-group membership
	return backends.PVE.AddGroupToPool(groupname, poolname)
}

func DelGroupFromPool(backends *UserSession, groupname common.Groupname, poolname string) (int, error) {
	// only pve backend handles pool-group membership
	return backends.PVE.DelGroupFromPool(groupname, poolname)
}

func NewUser(backends *UserSession, username common.Username, user common.User) (int, error) {
	if username.Realm == "pve" {
		return backends.PVE.NewUser(username, user)
	} else if username.Realm == backends.Realm.Name {
		realm_handler := backends.Realm.Handler.(common.Backend)
		code, err := realm_handler.NewUser(username, user)
		if err != nil {
			return code, err
		}
		return backends.PVE.SyncRealms()
	} else {
		return http.StatusUnauthorized, fmt.Errorf("user is not in the same realm as requested user")
	}
}

func GetUser(backends *UserSession, username common.Username) (common.User, int, error) {
	// fetch user from relevant realm
	if username.Realm == "pve" {
		pveuser, code, err := backends.PVE.GetUser(username)
		if err != nil {
			return common.User{}, code, err
		}
		return pveuser, http.StatusOK, nil
	} else if username.Realm == backends.Realm.Name {
		realmuser, code, err := backends.Realm.Handler.(common.Backend).GetUser(username)
		if err != nil {
			return common.User{}, code, err
		}
		return realmuser, http.StatusOK, nil
	} else {
		return common.User{}, http.StatusUnauthorized, fmt.Errorf("user is not in the same realm as requested user")
	}
}

func DelUser(backends *UserSession, username common.Username) (int, error) {
	if username.Realm == "pve" {
		return backends.PVE.DelUser(username)
	} else if username.Realm == backends.Realm.Name {
		realm_handler := backends.Realm.Handler.(common.Backend)
		code, err := realm_handler.DelUser(username)
		if err != nil {
			return code, err
		}
		return backends.PVE.SyncRealms()
	} else {
		return http.StatusUnauthorized, fmt.Errorf("user is not in the same realm as requested user")
	}
}

func AddUserToGroup(backends *UserSession, username common.Username, groupname common.Groupname) (int, error) {
	if username.Realm == "pve" && groupname.Realm == "pve" { // both req user and req group are in proxmox
		return backends.PVE.AddUserToGroup(username, groupname)
	} else if username.Realm == backends.Realm.Name && groupname.Realm == "pve" { // requested user in realm and req group in proxmox
		// this is a special case that is only supported because proxmox allows it
		// if user@realm is added to a pve group, then sync realm DOES NOT clear the group from the user
		// therefore adding user@realm to pve group should be allowed
		// in the future support may be removed
		return backends.PVE.AddUserToGroup(username, groupname)
	} else if username.Realm == backends.Realm.Name && groupname.Realm == backends.Realm.Name { // both req user and req group are in realm
		realm_handler := backends.Realm.Handler.(common.Backend)
		code, err := realm_handler.AddUserToGroup(username, groupname)
		if err != nil {
			return code, err
		}
		return backends.PVE.SyncRealms()
	} else { // req user in proxmox and req group in realm (not possible to do)
		return http.StatusUnauthorized, fmt.Errorf("cannot add %s to %s", username.ToString(), groupname.ToString())
	}
}

func DelUserFromGroup(backends *UserSession, username common.Username, groupname common.Groupname) (int, error) {
	if username.Realm == "pve" && groupname.Realm == "pve" { /// both req user and req group are in proxmox
		return backends.PVE.DelUserFromGroup(username, groupname)
	} else if username.Realm == backends.Realm.Name && groupname.Realm == "pve" { // requested user in realm and req group in proxmox
		// this is a special case that is only supported because proxmox allows it
		// if user@realm was added to a pve group, then sync realm DOES NOT clear the group from the user
		// therefore removing user@realm from pve group should be allowed
		// in the future support may be removed
		return backends.PVE.DelUserFromGroup(username, groupname)
	} else if username.Realm == backends.Realm.Name && groupname.Realm == backends.Realm.Name { // both req user and req group are in realm
		realm_handler := backends.Realm.Handler.(common.Backend)
		code, err := realm_handler.DelUserFromGroup(username, groupname)
		if err != nil {
			return code, err
		}
		return backends.PVE.SyncRealms()
	} else { // req user in proxmox and req group in realm (not possible to do)
		return http.StatusUnauthorized, fmt.Errorf("cannot delete %s from %s", username.ToString(), groupname.ToString())
	}
}
