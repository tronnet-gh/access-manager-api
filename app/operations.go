package app

import (
	common "user-manager-api/app/common"
)

func NewPool(backends *Backends, poolname string) (int, error) {
	return backends.pve.NewPool(poolname)
}
func DelPool(backends *Backends, poolname string) (int, error) {
	return backends.pve.DelPool(poolname)
}

func NewGroup(backends *Backends, groupname common.Groupname) (int, error) {
	handler := Config.Realms[groupname.Realm].Handler
	switch handler {
	case "pve":
		return backends.pve.NewGroup(groupname)
	case "ldap":

		code, err := backends.ldap.NewGroup(groupname)
		if err != nil {
			return code, err
		}

		//pve sync
		return backends.pve.SyncRealms()
	}
	return 200, nil
}

func DelGroup(backends *Backends, groupname common.Groupname) (int, error) {
	handler := Config.Realms[groupname.Realm].Handler
	switch handler {
	case "pve":
		return backends.pve.DelGroup(groupname)
	case "ldap":
		code, err := backends.ldap.DelGroup(groupname)
		if err != nil {
			return code, err
		}

		//pve sync
		return backends.pve.SyncRealms()
	}
	return 200, nil
}

func AddGroupToPool(backends *Backends, groupname common.Groupname, poolname string) (int, error) {
	// only pve backend handles pool-group membership
	return backends.pve.AddGroupToPool(groupname, poolname)
}

func DelGroupFromPool(backends *Backends, groupname common.Groupname, poolname string) (int, error) {
	// only pve backend handles pool-group membership
	return backends.pve.DelGroupFromPool(groupname, poolname)
}

func NewUser(backends *Backends, username common.Username, user common.User) (int, error) {
	handler := Config.Realms[username.Realm].Handler
	switch handler {
	case "pve":
		return backends.pve.NewUser(username, user)
	case "ldap":
		code, err := backends.ldap.NewUser(username, user)
		if err != nil {
			return code, err
		}

		//pve sync
		return backends.pve.SyncRealms()
	}
	return 200, nil
}

func DelUser(backends *Backends, username common.Username) (int, error) {
	handler := Config.Realms[username.Realm].Handler
	switch handler {
	case "pve":
		return backends.pve.DelUser(username)
	case "ldap":
		code, err := backends.ldap.DelUser(username)
		if err != nil {
			return code, err
		}

		//pve sync
		return backends.pve.SyncRealms()
	}
	return 200, nil
}

func AddUserToGroup(backends *Backends, username common.Username, groupname common.Groupname) (int, error) {
	handler := Config.Realms[username.Realm].Handler
	switch handler {
	case "pve":
		return backends.pve.AddUserToGroup(username, groupname)
	case "ldap":
		code, err := backends.ldap.AddUserToGroup(username, groupname)
		if err != nil {
			return code, err
		}

		//pve sync
		return backends.pve.SyncRealms()
	}
	return 200, nil
}

func DelUserFromGroup(backends *Backends, username common.Username, groupname common.Groupname) (int, error) {
	handler := Config.Realms[username.Realm].Handler
	switch handler {
	case "pve":
		return backends.pve.DelUserFromGroup(username, groupname)
	case "ldap":
		code, err := backends.ldap.DelUserFromGroup(username, groupname)
		if err != nil {
			return code, err
		}

		//pve sync
		return backends.pve.SyncRealms()
	}
	return 200, nil
}
