package app

import paas "proxmoxaas-common-lib"

type Backend interface {
	NewPool(poolname string) (int, error)
	DelPool(poolname string) (int, error)
	NewGroup(groupname Groupname) (int, error)
	DelGroup(groupname Groupname) (int, error)
	AddGroupToPool(groupname Groupname, poolname string) (int, error)
	DelGroupFromPool(groupname Groupname, poolname string) (int, error)
	NewUser(username Username, user User) (int, error)
	DelUser(username Username) (int, error)
	AddUserToGroup(username Username, groupname Groupname) (int, error)
	DelUserFromGroup(username Username, groupname Groupname) (int, error)
}

type Pool = paas.Pool
type Groupname = paas.Groupname
type Group = paas.Group
type Username = paas.Username
type User = paas.User
type VMID = paas.VMID
type Backups = paas.Backups
type Templates = paas.Templates
type SimpleResource = paas.SimpleResource
type SimpleLimit = paas.SimpleLimit
type MatchResource = paas.MatchResource
type MatchLimit = paas.MatchLimit
type ResourceTemplate = paas.ResourceTemplate
