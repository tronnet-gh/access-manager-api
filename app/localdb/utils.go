package localdb

import (
	common "access-manager-api/app/common"
	"encoding/json"
	"os"
	"reflect"
)

func (db *DB) load(localDBPath string) error {
	db.data = make(map[string]common.Pool)
	db.path = localDBPath

	root, err := os.OpenRoot(".")
	if err != nil {
		return err
	}
	defer root.Close()

	content, err := root.ReadFile(localDBPath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(content, &db.data)
	if err != nil {
		return err
	}
	return nil
}

func (db *DB) save() error {
	localDBPath := db.path
	// write to file with pretty print for readability reasons
	json, err := json.MarshalIndent(db.data, "", "\t")
	if err != nil {
		return err
	}
	err = os.WriteFile(localDBPath, []byte(json), 0600)
	return err
}

// MergeNonZero overwrites fields in dst with fields in src iff src is not a zero value.
func MergeNonZero[T any](dst *T, src *T) {
	vDst := reflect.ValueOf(dst).Elem()
	vSrc := reflect.ValueOf(src).Elem()

	for i := 0; i < vDst.NumField(); i++ {
		dstField := vDst.Field(i)
		srcField := vSrc.Field(i)

		// Only set if the field is exported (CanSet) and the source is not a zero value
		if dstField.CanSet() && !srcField.IsZero() {
			dstField.Set(srcField)
		}
	}
}
