package main

import (
	"flag"
	app "user-manager-api/app"
)

func main() {
	configPath := flag.String("config", "config.json", "path to config.json file")
	localDBPath := flag.String("localdb", "localdb.json", "path to localdb.json file")
	flag.Parse()
	app.Run(configPath, localDBPath)
}
