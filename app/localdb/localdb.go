package localdb

import (
	common "access-manager-api/app/common"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type DB struct {
	data map[string]common.Pool
}

func LoadDB(localDBPath string) (DB, error) {
	db := DB{}

	root, err := os.OpenRoot(".")
	if err != nil {
		return db, err
	}
	defer root.Close()

	content, err := root.ReadFile(localDBPath)
	if err != nil {
		return db, err
	}

	err = json.Unmarshal(content, &db.data)
	if err != nil {
		return db, err
	}
	return db, nil
}

func SaveDB(localDBPath string, db DB) error {
	json, err := json.Marshal(db.data)
	if err != nil {
		return err
	}
	err = os.WriteFile(localDBPath, []byte(json), 0600)
	return err
}

func (localdb DB) GetPool(poolname string) (common.Pool, []string, int, error) {
	pool, ok := localdb.data[poolname]
	if !ok {
		return pool, []string{}, http.StatusNotFound, fmt.Errorf("pool %s not in localdb", poolname)
	}
	return pool, []string{}, http.StatusOK, nil
}
