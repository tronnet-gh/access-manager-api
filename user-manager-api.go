package main

import (
	"flag"
	app "user-manager-api/app"
)

func main() {
	configPath := flag.String("config", "config.json", "path to config.json file")
	flag.Parse()
	app.Run(configPath)
}
