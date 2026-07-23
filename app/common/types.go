package app

import paas "proxmoxaas-common-lib"

type Backend interface {
	NewPool(poolname string) (int, error)
	GetPool(poolname string) (Pool, []string, int, error) // []string members
	DelPool(poolname string) (int, error)
	NewGroup(groupname Groupname) (int, error)
	GetGroup(groupname Groupname) (Group, []string, int, error) // []string members
	DelGroup(groupname Groupname) (int, error)
	AddGroupToPool(groupname Groupname, poolname string) (int, error)
	DelGroupFromPool(groupname Groupname, poolname string) (int, error)
	NewUser(username Username, user User) (int, error)
	GetUser(username Username) (User, int, error)
	DelUser(username Username) (int, error)
	AddUserToGroup(username Username, groupname Groupname) (int, error)
	DelUserFromGroup(username Username, groupname Groupname) (int, error)
}

type Pool = paas.Pool
type Groupname = paas.Groupname
type Group = paas.Group
type Username = paas.Username
type User = paas.User

func ParseGroupname(groupname string) (Groupname, error) {
	return paas.ParseGroupname(groupname)
}

func ParseUsername(username string) (Username, error) {
	return paas.ParseUsername(username)
}
