package localdb

import (
	common "access-manager-api/app/common"
	"fmt"
	"net/http"
)

type DB struct {
	path string
	data map[string]common.Pool
}

var db *DB

func NewClientFromCredentials(config common.LocalDBConfig, username common.Username, password string) (common.Backend, int, error) {
	if db != nil {
		return *db, http.StatusOK, nil
	} else {
		// load localdb if this is the first time
		db = &DB{}
		err := db.load(config.Path)
		if err != nil {
			return *db, http.StatusInternalServerError, err
		} else {
			return *db, http.StatusOK, nil
		}
	}
}

func (localdb DB) GetPool(poolname string) (common.Pool, []string, int, error) {
	pool, ok := localdb.data[poolname]
	if !ok {
		return pool, []string{}, http.StatusNotFound, fmt.Errorf("localdb pool %s does not exist", poolname)
	}
	return pool, []string{}, http.StatusOK, nil
}

func (localdb DB) NewPool(poolname string, pool common.Pool) (int, error) {
	_, ok := localdb.data[poolname]
	if ok {
		return http.StatusBadRequest, fmt.Errorf("localdb pool %s already exists", poolname)
	}
	localdb.data[poolname] = pool
	err := localdb.save()
	if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (localdb DB) ModPool(poolname string, pool common.Pool) (int, error) {
	_, ok := localdb.data[poolname]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("localdb pool %s does not exist", poolname)
	}
	old_pool := localdb.data[poolname]
	MergeNonZero(&old_pool, &pool)
	err := localdb.save()
	if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (localdb DB) DelPool(poolname string) (int, error) {
	_, ok := localdb.data[poolname]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("localdb pool %s does not exist", poolname)
	}
	delete(localdb.data, poolname)
	err := localdb.save()
	if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (localdb DB) NewGroup(groupname common.Groupname, group common.Group) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement groups")
}

func (localdb DB) ModGroup(groupname common.Groupname, group common.Group) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement groups")
}

func (localdb DB) GetGroup(groupname common.Groupname) (common.Group, []string, int, error) {
	return common.Group{}, []string{}, http.StatusNotImplemented, fmt.Errorf("localdb does not implement groups")
}

func (localdb DB) DelGroup(groupname common.Groupname) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement groups")
}

func (localdb DB) AddGroupToPool(groupname common.Groupname, poolname string) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement groups")
}

func (localdb DB) DelGroupFromPool(groupname common.Groupname, poolname string) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement groups")
}

func (localdb DB) NewUser(username common.Username, user common.User) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement users")
}

func (localdb DB) ModUser(username common.Username, user common.User) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement users")
}

func (localdb DB) GetUser(username common.Username) (common.User, int, error) {
	return common.User{}, http.StatusNotImplemented, fmt.Errorf("localdb does not implement users")
}

func (localdb DB) DelUser(username common.Username) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement users")
}

func (localdb DB) AddUserToGroup(username common.Username, groupname common.Groupname) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement users")
}

func (localdb DB) DelUserFromGroup(username common.Username, groupname common.Groupname) (int, error) {
	return http.StatusNotImplemented, fmt.Errorf("localdb does not implement users")
}
