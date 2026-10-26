package app

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	paas "proxmoxaas-common-lib"

	"github.com/luthermonson/go-proxmox"
)

const APIVersion string = "1.0.0"

var client ProxmoxClient

func Run() {
	gob.Register(proxmox.Client{})

	configPath := flag.String("config", "config.json", "path to config.json file")
	flag.Parse()

	config := GetConfig(*configPath)
	log.Printf("[INFO] initialized config from %s", *configPath)

	token := fmt.Sprintf(`%s@%s!%s`, config.PVE.Token.User, config.PVE.Token.Realm, config.PVE.Token.ID)
	client = NewClient(config.PVE.URL, token, config.PVE.Token.Secret)

	router := http.NewServeMux()

	cluster := Cluster{}
	cluster.Init(client)
	log.Printf("[INFO] starting cluster sync\n")
	err := cluster.Sync()
	if err != nil {
		log.Printf("[ERR ] error encountered while syncing cluster: %s", err)
	}

	// set repeating update for full rebuilds
	ticker := time.NewTicker(time.Duration(config.ReloadInterval) * time.Second)
	log.Printf("[INFO] initialized cluster sync interval of %ds", config.ReloadInterval)
	channel := make(chan bool)
	go func() {
		for {
			select {
			case <-channel:
				return
			case <-ticker.C:
				log.Printf("[INFO] starting cluster sync\n")
				err := cluster.Sync()
				if err != nil {
					log.Printf("[ERR ] error encountered while syncing cluster: %s", err)
				}
			}
		}
	}()

	router.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		PVEVersion, err := client.Version()
		if err != nil {
			paas.HTTPWriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		} else {
			paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"api-version": APIVersion, "pve-version": PVEVersion})
		}
	})

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		callback := Callback{}
		v, err := GetCluster(&callback, &cluster)
		if err != nil {
			paas.HTTPWriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		} else {
			paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"cluster": v})
		}
		callback.Invoke()
	})

	router.HandleFunc("GET /nodes/{node}", func(w http.ResponseWriter, r *http.Request) {
		nodeid := r.PathValue("node")
		callback := Callback{}
		node, err := GetNode(&callback, &cluster, nodeid)
		if err != nil {
			paas.HTTPWriteJson(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		} else {
			paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"node": node})
		}
		callback.Invoke()
	})

	router.HandleFunc("GET /nodes/{node}/instances/{vmid}", func(w http.ResponseWriter, r *http.Request) {
		nodeid := r.PathValue("node")
		vmid, err := strconv.ParseUint(r.PathValue("vmid"), 10, 64)
		if err != nil {
			paas.HTTPWriteJson(w, http.StatusBadRequest, map[string]any{"error": fmt.Sprintf("%s could not be converted to vmid (uint)", r.PathValue("instance"))})
			return
		}
		callback := Callback{}
		instance, err := GetInstance(&callback, &cluster, nodeid, vmid)
		if err != nil {
			paas.HTTPWriteJson(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		} else {
			paas.HTTPWriteJson(w, http.StatusOK, map[string]any{"instance": instance})
		}
		callback.Invoke()
	})

	router.HandleFunc("POST /sync", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[INFO] starting cluster sync\n")
		err := cluster.Sync()
		if err != nil {
			log.Printf("[ERR ] failed to sync cluster: %s", err)
			paas.HTTPWriteJson(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		} else {
			return
		}
	})

	router.HandleFunc("POST /nodes/{node}/sync", func(w http.ResponseWriter, r *http.Request) {
		nodeName := r.PathValue("node")

		log.Printf("[INFO] starting %s sync\n", nodeName)

		callback := Callback{}
		node, err := GetNode(&callback, &cluster, nodeName)
		if err != nil {
			log.Printf("[ERR ] failed to sync %s: %s", nodeName, err.Error())
			paas.HTTPWriteJson(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			callback.Invoke()
			return
		}
		callback.Invoke()

		err = node.Sync()
		if err != nil {
			log.Printf("[ERR ] failed to sync %s: %s", nodeName, err.Error())
			paas.HTTPWriteJson(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		}
	})

	router.HandleFunc("POST /nodes/{node}/instances/{vmid}/sync", func(w http.ResponseWriter, r *http.Request) {
		nodeName := r.PathValue("node")
		vmid, err := strconv.ParseUint(r.PathValue("vmid"), 10, 64)
		if err != nil {
			log.Printf("[ERR ] failed to sync %s.%d: %s", nodeName, vmid, err.Error())
			paas.HTTPWriteJson(w, http.StatusInternalServerError, map[string]any{"error": fmt.Sprintf("%s could not be converted to vmid (uint)", r.PathValue("instance"))})
			return
		}

		log.Printf("[INFO] starting %s.%d sync\n", nodeName, vmid)

		callback := Callback{}
		instance, err := GetInstance(&callback, &cluster, nodeName, vmid)
		if err != nil {
			log.Printf("[ERR ] failed to sync %s.%d: %s", nodeName, vmid, err.Error())
			paas.HTTPWriteJson(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			callback.Invoke()
			return
		}
		callback.Invoke()

		err = instance.Sync()
		if err != nil {
			log.Printf("[ERR ] failed to sync %s.%d: %s", nodeName, vmid, err.Error())
			paas.HTTPWriteJson(w, http.StatusNotFound, map[string]any{"error": err.Error()})
		}
	})

	server := paas.LoggingMiddleware(router)

	log.Printf("[INFO] starting API listening on 0.0.0.0:%d", config.ListenPort)
	log.Fatalf("[ERR ] starting gin router: %s", http.ListenAndServe("0.0.0.0:"+strconv.Itoa(config.ListenPort), server))
}
