package localdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	common "user-manager-api/app/common"
)

type DB struct {
	data map[string]common.Pool
}

func LoadDB(localDBPath string) (DB, error) {
	db := DB{}
	content, err := os.ReadFile(localDBPath)
	if err != nil {
		//log.Fatal("Error when opening file: ", err)
		return db, err
	}
	err = json.Unmarshal(content, &db.data)
	if err != nil {
		//log.Fatal("Error during Unmarshal(): ", err)
		return db, err
	}
	return db, nil
}

func SaveDB(localDBPath string, db DB) error {
	json, err := json.Marshal(db.data)
	if err != nil {
		return err
	}
	err = os.WriteFile(localDBPath, []byte(json), 0644)
	return err
}

func (localdb DB) GetPool(poolname string) (common.Pool, []string, int, error) {
	pool := common.Pool{}
	pool, ok := localdb.data[poolname]
	if !ok {
		return pool, []string{}, http.StatusNotFound, fmt.Errorf("Pool %s not in localdb", poolname)
	}
	return pool, []string{}, http.StatusOK, nil
}
