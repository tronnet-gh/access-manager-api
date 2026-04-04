package app

import (
	"fmt"
	"net/http"
	common "user-manager-api/app/common"
)

func NewPool(backends *UserSession, poolname string) (int, error) {
	// only pve backend handles pools
	return backends.PVE.NewPool(poolname)
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
	if username.Realm == "pve" && groupname.Realm == "pve" { // both requested user and requested group are in proxmox
		return backends.PVE.AddUserToGroup(username, groupname)
	} else if username.Realm == backends.Realm.Name && groupname.Realm == "pve" { // requested user is in user's realm but group is in proxmox
		return backends.PVE.AddUserToGroup(username, groupname)
	} else if username.Realm == backends.Realm.Name && groupname.Realm == backends.Realm.Name { // both requested user and requested group are in user's realm
		realm_handler := backends.Realm.Handler.(common.Backend)
		code, err := realm_handler.AddUserToGroup(username, groupname)
		if err != nil {
			return code, err
		}
		return backends.PVE.SyncRealms()
	} else {
		return http.StatusUnauthorized, fmt.Errorf("cannot add a pve user to a group in %s", groupname.Realm)
	}
}

func DelUserFromGroup(backends *UserSession, username common.Username, groupname common.Groupname) (int, error) {
	if username.Realm == "pve" && groupname.Realm == "pve" { // both requested user and requested group are in proxmox
		return backends.PVE.DelUserFromGroup(username, groupname)
	} else if username.Realm == backends.Realm.Name && groupname.Realm == "pve" { // requested user is in user's realm but group is in proxmox
		return backends.PVE.DelUserFromGroup(username, groupname)
	} else if username.Realm == backends.Realm.Name && groupname.Realm == backends.Realm.Name { // both requested user and requested group are in user's realm
		realm_handler := backends.Realm.Handler.(common.Backend)
		code, err := realm_handler.DelUserFromGroup(username, groupname)
		if err != nil {
			return code, err
		}
		return backends.PVE.SyncRealms()
	} else {
		return http.StatusUnauthorized, fmt.Errorf("cannot remove a pve user from a group in %s", groupname.Realm)
	}
}
