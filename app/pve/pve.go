package pve

import (
	"context"
	"crypto/tls"
	"net/http"

	common "user-manager-api/app/common"

	"github.com/luthermonson/go-proxmox"
)

type ProxmoxClient struct {
	client *proxmox.Client
}

func NewClientFromCredentials(config common.PVEConfig, username common.Username, password string) (*ProxmoxClient, int, error) {
	HTTPClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	client := proxmox.NewClient(config.URL,
		proxmox.WithHTTPClient(&HTTPClient),
		proxmox.WithCredentials(&proxmox.Credentials{Username: username.ToString(), Password: password}),
	)

	// todo this should return an error code if the binding failed (ie fetch version to check if the auth was actually ok)

	return &ProxmoxClient{client: client}, http.StatusOK, nil
}

func (pve ProxmoxClient) SyncRealms() (int, error) {
	domains, err := pve.client.Domains(context.Background())
	if proxmox.IsNotAuthorized(err) {
		return 401, err
	} else if err != nil {
		return 500, err
	}
	for _, domain := range domains {
		if domain.Type != "pam" && domain.Type != "pve" { // pam and pve are not external realm types that require sync
			err := domain.Sync(context.Background(), proxmox.DomainSyncOptions{
				DryRun:         false,                  // we want to make modifications
				EnableNew:      true,                   // allow new users and groups
				Scope:          "both",                 // allow new users and groups
				RemoveVanished: "acl;entry;properties", // remove deleted objects from ACL, entry in pve, and remove properties (probably not necessary)
			})
			if proxmox.IsNotAuthorized(err) {
				return 401, err
			} else if err != nil {
				return 500, err
			}
		}
	}
	return 200, nil
}

func (pve ProxmoxClient) NewPool(poolname string) (int, error) {
	err := pve.client.NewPool(context.Background(), poolname, "")
	if proxmox.IsNotAuthorized(err) {
		return 401, err
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}

func (pve ProxmoxClient) DelPool(poolname string) (int, error) {
	pvepool, err := pve.client.Pool(context.Background(), poolname)
	if proxmox.IsNotFound(err) { // errors if pool does not exist
		return 404, err
	} else if err != nil {
		return 500, err
	}

	err = pvepool.Delete(context.Background())
	if proxmox.IsNotAuthorized(err) { // not authorized to delete
		return 401, err
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}

func (pve ProxmoxClient) NewGroup(groupname common.Groupname) (int, error) {
	err := pve.client.NewGroup(context.Background(), groupname.ToString(), "")
	if proxmox.IsNotAuthorized(err) {
		return 401, err
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}

func (pve ProxmoxClient) DelGroup(groupname common.Groupname) (int, error) {
	pvegroup, err := pve.client.Group(context.Background(), groupname.ToString())
	if proxmox.IsNotFound(err) { // errors if group does not exist
		return 404, err
	} else if err != nil {
		return 500, err
	}

	err = pvegroup.Delete(context.Background())
	if proxmox.IsNotAuthorized(err) { // not authorized to delete
		return 401, err
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}
