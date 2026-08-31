package app

import paas "proxmoxaas-common-lib"

type Realm struct {
	Type   string
	Config any
}

type Backend interface {
	NewPool(poolname string, pool Pool) (int, error)
	ModPool(poolname string, pool Pool) (int, error)
	GetPool(poolname string) (Pool, []string, int, error) // []string members
	DelPool(poolname string) (int, error)
	NewGroup(groupname Groupname, group Group) (int, error)
	ModGroup(groupname Groupname, group Group) (int, error)
	GetGroup(groupname Groupname) (Group, []string, int, error) // []string members
	DelGroup(groupname Groupname) (int, error)
	AddGroupToPool(groupname Groupname, poolname string) (int, error)
	DelGroupFromPool(groupname Groupname, poolname string) (int, error)
	NewUser(username Username, user User) (int, error)
	ModUser(username Username, user User) (int, error)
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

type PVEAPIToken struct {
	User  string `json:"user"`
	Realm string `json:"realm"`
	ID    string `json:"id"`
	UUID  string `json:"uuid"`
}
