package main

import (
	"encoding/json"
	"log"
	"os"
)

func GetLocaldb(dbPath string) map[string]User {
	users := map[string]User{}
	content, err := os.ReadFile(dbPath)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}
	err = json.Unmarshal(content, &users)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}
	return users
}
