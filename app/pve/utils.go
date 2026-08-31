package pve

import (
	"strings"

	"github.com/luthermonson/go-proxmox"
)

func IsProxmoxNotFound(err error) bool {
	if err != nil {
		// for whatever reason proxmox returns 500 for user/group/pool not found
		return proxmox.IsNotFound(err) || strings.Contains(err.Error(), "no such") || strings.Contains(err.Error(), "does not exist")
	}
	return false
}
