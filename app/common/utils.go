package app

import (
	"fmt"
	"reflect"

	paas "proxmoxaas-common-lib"
)

func ParseGroupname(groupname string) (Groupname, error) {
	return paas.ParseGroupname(groupname)
}

func ParseUsername(username string) (Username, error) {
	return paas.ParseUsername(username)
}

// RequireAll ensures that EVERY non-excluded exported field in the struct is non-zero.
func RequireAll(v any, excludes ...string) bool {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	typ := val.Type()

	excludeMap := make(map[string]bool)
	for _, ex := range excludes {
		excludeMap[ex] = true
	}

	for i := 0; i < val.NumField(); i++ {
		fieldName := typ.Field(i).Name
		if excludeMap[fieldName] {
			continue
		}
		if val.Field(i).IsZero() {
			return false
		}
	}
	return true
}

// AtLeastOne ensures that AT LEAST ONE non-excluded exported field is non-zero.
func AtLeastOne(v any, excludes ...string) bool {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	typ := val.Type()

	excludeMap := make(map[string]bool)
	for _, ex := range excludes {
		excludeMap[ex] = true
	}

	for i := 0; i < val.NumField(); i++ {
		fieldName := typ.Field(i).Name
		if excludeMap[fieldName] {
			continue
		}
		if !val.Field(i).IsZero() {
			return true
		}
	}
	return false
}

func (token PVEAPIToken) ToString() string {
	return fmt.Sprintf(`%s@%s!%s`, token.User, token.Realm, token.ID)
}
