package pve

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"slices"

	common "user-manager-api/app/common"

	"github.com/luthermonson/go-proxmox"
)

type ProxmoxClient struct {
	config *common.PVEConfig
	client *proxmox.Client
}

// creates a new client binding with associated permissions
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
		proxmox.WithCredentials(&proxmox.Credentials{Username: username.UserID, Realm: username.Realm, Password: password}),
	)

	// check that the user is authenticated because proxmox.NewClient does not return an error
	// version route is accessible to any authenticated user
	_, err := client.Version(context.Background())
	if err != nil { // could not get version so therefore the user is not authenticated
		return nil, http.StatusUnauthorized, err
	}

	// todo this should return an error code if the binding failed (ie fetch version to check if the auth was actually ok)
	return &ProxmoxClient{config: &config, client: client}, http.StatusOK, nil
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

func (pve ProxmoxClient) AddGroupToPool(groupname common.Groupname, poolname string) (int, error) {
	// adds the group to the pool with the predetermined PAAS client role
	err := pve.client.UpdateACL(context.Background(), proxmox.ACLOptions{
		Path:      fmt.Sprintf("/pool/%s", poolname),
		Groups:    groupname.ToString(),
		Roles:     pve.config.PAASClientRole,
		Propagate: true,
	})

	if proxmox.IsNotAuthorized(err) {
		return 401, err
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}

func (pve ProxmoxClient) DelGroupFromPool(groupname common.Groupname, poolname string) (int, error) {
	// removes the group from the pool with the predetermined PAAS client role
	err := pve.client.UpdateACL(context.Background(), proxmox.ACLOptions{
		Path:   fmt.Sprintf("/pool/%s", poolname),
		Groups: groupname.ToString(),
		Roles:  pve.config.PAASClientRole,
		Delete: true,
	})

	if proxmox.IsNotAuthorized(err) {
		return 401, err
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}

func (pve ProxmoxClient) NewUser(username common.Username, user common.User) (int, error) {
	err := pve.client.NewUser(context.Background(), &proxmox.NewUser{
		UserID:    username.ToString(),
		Firstname: user.CN,
		Lastname:  user.SN,
		Email:     user.Mail,
		Password:  user.Password,
	})
	if proxmox.IsNotAuthorized(err) {
		return 401, err
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}

func (pve ProxmoxClient) DelUser(username common.Username) (int, error) {
	user, err := pve.client.User(context.Background(), username.ToString())
	if proxmox.IsNotAuthorized(err) {
		return 401, err // not authorized to read the user = not authorized to delete the user, this may be slightly different from ldap
	} else if err != nil {
		return 500, err
	}

	// assume that user cannot be nil if no error was returned
	err = user.Delete(context.Background())
	if proxmox.IsNotAuthorized(err) {
		return 401, err // not authorized to delete the user
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}

func (pve ProxmoxClient) AddUserToGroup(username common.Username, groupname common.Groupname) (int, error) {
	user, err := pve.client.User(context.Background(), groupname.ToString())
	if proxmox.IsNotAuthorized(err) {
		return 401, err // not authorized to read the user = not authorized to delete the user, this may be slightly different from ldap
	} else if err != nil {
		return 500, err
	}

	newGroups := append(user.Groups, groupname.ToString())

	err = user.Update(context.Background(), proxmox.UserOptions{
		Groups: newGroups,
	})
	if proxmox.IsNotAuthorized(err) {
		return 401, err // not authorized to delete the user
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}

func (pve ProxmoxClient) DelUserFromGroup(username common.Username, groupname common.Groupname) (int, error) {
	user, err := pve.client.User(context.Background(), groupname.ToString())
	if proxmox.IsNotAuthorized(err) {
		return 401, err // not authorized to read the user = not authorized to delete the user, this may be slightly different from ldap
	} else if err != nil {
		return 500, err
	}

	idx := slices.Index(user.Groups, groupname.ToString())
	if idx < 0 {
		return http.StatusBadRequest, fmt.Errorf("Did not find group %s in user groups {%+v}.", groupname.ToString(), user.Groups)
	}
	newGroups := slices.Delete(user.Groups, idx, idx)

	err = user.Update(context.Background(), proxmox.UserOptions{
		Groups: newGroups,
	})
	if proxmox.IsNotAuthorized(err) {
		return 401, err // not authorized to delete the user
	} else if err != nil {
		return 500, err
	} else {
		return 200, nil
	}
}
