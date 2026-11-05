package app

import paas "access-manager-api/proxmoxaas-common-lib"

type Realm struct {
	Type   string
	Config any
}

type Backend interface {
	NewPool(poolname Poolname, pool Pool) (int, error)
	ModPool(poolname Poolname, pool Pool) (int, error)
	GetPool(poolname Poolname) (Pool, []Group, int, error) // []string members
	GetAllPoolnames() ([]Poolname, int, error)
	DelPool(poolname Poolname) (int, error)
	NewGroup(groupname Groupname, group Group) (int, error)
	ModGroup(groupname Groupname, group Group) (int, error)
	GetGroup(groupname Groupname) (Group, []User, int, error) // []string members
	GetAllGroupnames() ([]Groupname, int, error)
	DelGroup(groupname Groupname) (int, error)
	AddGroupToPool(groupname Groupname, poolname Poolname) (int, error)
	DelGroupFromPool(groupname Groupname, poolname Poolname) (int, error)
	NewUser(username Username, user User) (int, error)
	ModUser(username Username, user User) (int, error)
	GetUser(username Username) (User, int, error)
	GetAllUsernames() ([]Username, int, error)
	DelUser(username Username) (int, error)
	AddUserToGroup(username Username, groupname Groupname) (int, error)
	DelUserFromGroup(username Username, groupname Groupname) (int, error)
}

type Poolname = string
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
