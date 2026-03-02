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
		backends.ldap.NewGroup(groupname)
		//pve sync
		return 200, nil
	}
	return 200, nil
}

func DelGroup(backends *Backends, groupname common.Groupname) (int, error) {
	handler := Config.Realms[groupname.Realm].Handler
	switch handler {
	case "pve":
		return backends.pve.DelGroup(groupname)
	case "ldap":
		backends.ldap.DelGroup(groupname)
		//pve sync
		return 200, nil
	}
	return 200, nil
}
