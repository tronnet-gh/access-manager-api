package pve

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"slices"

	common "access-manager-api/app/common"

	"github.com/luthermonson/go-proxmox"
)

type ProxmoxClient struct {
	config *common.PVEConfig
	client *proxmox.Client
}

// creates a new client binding with associated permissions
func NewClientFromCredentials(config common.PVEConfig, username common.Username, password string) (common.Backend, int, error) {
	HTTPClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{},
		},
	}

	client := proxmox.NewClient(config.URL,
		proxmox.WithHTTPClient(&HTTPClient),
		proxmox.WithCredentials(&proxmox.Credentials{Username: username.UserID, Realm: username.Realm, Password: password}),
		proxmox.WithEagerAuth(),
	)

	// check that the user is authenticated because proxmox.NewClient does not return an error
	// version route is accessible to any authenticated user
	_, err := client.Version(context.Background())
	if err != nil { // could not get version so therefore the user is not authenticated
		return nil, http.StatusUnauthorized, err
	}

	return ProxmoxClient{config: &config, client: client}, http.StatusOK, nil
}

// creates a new client binding with associated permissions
func NewClientFromAPIToken(config common.PVEConfig, token common.PVEAPIToken) (common.Backend, int, error) {
	HTTPClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{},
		},
	}
	client := proxmox.NewClient(config.URL,
		proxmox.WithHTTPClient(&HTTPClient),
		proxmox.WithAPIToken(token.ToString(), config.Token.UUID),
	)

	// check that the user is authenticated because proxmox.NewClient does not return an error
	// version route is accessible to any authenticated user
	_, err := client.Version(context.Background())
	if err != nil { // could not get version so therefore the user is not authenticated
		return nil, http.StatusUnauthorized, err
	}

	return ProxmoxClient{config: &config, client: client}, http.StatusOK, nil
}

func (pve ProxmoxClient) SyncRealms() (int, error) {
	domains, err := pve.client.Domains(context.Background())
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}
	for _, domain := range domains {
		if domain.Type != "pam" && domain.Type != "pve" { // pam and pve are not external realm types that require sync
			e := proxmox.IntOrBool(true)
			r := "acl;entry;properties"
			err := domain.Sync(context.Background(), proxmox.DomainSyncOptions{
				DryRun:         false,  // we want to make modifications
				EnableNew:      &e,     // allow new users and groups
				Scope:          "both", // allow new users and groups
				RemoveVanished: &r,     // remove deleted objects from ACL, entry in pve, and remove properties (probably not necessary)
			})
			if proxmox.IsNotAuthorized(err) {
				return http.StatusUnauthorized, err
			} else if err != nil {
				return http.StatusInternalServerError, err
			}
		}
	}
	return http.StatusOK, nil
}

func (pve ProxmoxClient) GetRealms() map[string]common.Realm {
	realms := map[string]common.Realm{}

	pverealms, err := pve.client.Domains(context.Background())
	if err != nil {
		// failure to get realms is a fatal error
		log.Fatalf("Error getting authentication realms: %s", err.Error())
	}

	// add required pve realm handler, removing the pve api token
	pveconfig := common.PVEConfig{
		URL:            pve.config.URL,
		PAASClientRole: pve.config.PAASClientRole,
	}
	realms["pve"] = common.Realm{
		Type:   "pve",
		Config: pveconfig,
	}
	log.Printf("Configured default authentication realm pve")

	// iterate through handlers and add to realms
	for _, r := range pverealms {
		realm, err := pve.client.Domain(context.Background(), r.Realm)
		if err != nil {
			log.Printf("Error getting authentication realm %s: %s", r.Realm, err.Error())
		}

		if realm.Type == "ldap" {
			ldapconfig := common.LDAPConfig{
				BaseDN:   realm.BaseDN,
				Hostname: realm.Server1,
				TLS:      realm.Mode == "ldaps",
				StartTLS: realm.Mode == "ldap+starttls",
				Verify:   bool(realm.Verify),
			}
			realms[realm.Realm] = common.Realm{
				Type:   realm.Type,
				Config: ldapconfig,
			}
			log.Printf("Configured external authentication realm %s", realm.Realm)
		} else {
			continue
		}
	}

	return realms
}

func (pve ProxmoxClient) NewPool(poolname string, pool common.Pool) (int, error) {
	err := pve.client.NewPool(context.Background(), poolname, "")
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (pve ProxmoxClient) ModPool(poolname string, pool common.Pool) (int, error) {
	// no-op
	return http.StatusOK, nil
}

func (pve ProxmoxClient) GetPool(poolname string) (common.Pool, []string, int, error) {
	pool := common.Pool{}
	members := []string{}

	pvepool, err := pve.client.Pool(context.Background(), poolname)
	if IsProxmoxNotFound(err) { // errors if pool does not exist
		return pool, members, http.StatusNotFound, err
	} else if err != nil {
		return pool, members, http.StatusInternalServerError, err
	}

	pool.PoolID = pvepool.PoolID

	acls, err := pve.client.ACL(context.Background())
	if proxmox.IsNotAuthorized(err) { // errors if pool does not exist
		return pool, members, http.StatusUnauthorized, err
	} else if err != nil {
		return pool, members, http.StatusInternalServerError, err
	}

	// iterate through ACLs and get superficial pool - group membershipm (just group names)
	for _, acl := range acls {
		if acl.Type == "group" && acl.RoleID == pve.config.PAASClientRole && acl.Path == fmt.Sprintf("/pool/%s", poolname) {
			members = append(members, acl.UGID)
		}
	}

	return pool, members, http.StatusOK, nil
}

func (pve ProxmoxClient) DelPool(poolname string) (int, error) {
	pvepool, err := pve.client.Pool(context.Background(), poolname)
	if proxmox.IsNotAuthorized(err) { // not authorized to delete
		return http.StatusUnauthorized, err
	} else if IsProxmoxNotFound(err) { // errors if pool does not exist
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	err = pvepool.Delete(context.Background())
	if proxmox.IsNotAuthorized(err) { // not authorized to delete
		return http.StatusUnauthorized, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (pve ProxmoxClient) NewGroup(groupname common.Groupname, group common.Group) (int, error) {
	// add new group by ID only
	err := pve.client.NewGroup(context.Background(), groupname.GroupID, "")
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (pve ProxmoxClient) ModGroup(groupname common.Groupname, group common.Group) (int, error) {
	// no-op
	return http.StatusOK, nil
}

func (pve ProxmoxClient) GetGroup(groupname common.Groupname) (common.Group, []string, int, error) {
	group := common.Group{}
	members := []string{}
	pvegroup, err := pve.client.Group(context.Background(), groupname.ToString())
	if IsProxmoxNotFound(err) { // errors if pool does not exist
		return group, members, http.StatusNotFound, err
	} else if err != nil {
		return group, members, http.StatusInternalServerError, err
	}

	group.Groupname, _ = common.ParseGroupname(pvegroup.GroupID)

	members = append(members, pvegroup.Members...)

	return group, members, http.StatusOK, nil
}

func (pve ProxmoxClient) DelGroup(groupname common.Groupname) (int, error) {
	pvegroup, err := pve.client.Group(context.Background(), groupname.GroupID)
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if IsProxmoxNotFound(err) {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	err = pvegroup.Delete(context.Background())
	if proxmox.IsNotAuthorized(err) { // not authorized to delete
		return http.StatusUnauthorized, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
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
		return http.StatusUnauthorized, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
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
		return http.StatusUnauthorized, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (pve ProxmoxClient) NewUser(username common.Username, user common.User) (int, error) {
	if !common.RequireAll(user, "Username") {
		return http.StatusBadRequest, fmt.Errorf("missing one of required fields: cn, sn, mail, userpassword")
	}

	err := pve.client.NewUser(context.Background(), &proxmox.NewUser{
		UserID:    username.ToString(),
		Firstname: user.CN,
		Lastname:  user.SN,
		Email:     user.Mail,
		Password:  user.Password,
	})
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (pve ProxmoxClient) ModUser(username common.Username, user common.User) (int, error) {
	if !common.AtLeastOne(user, "Username") {
		return http.StatusBadRequest, fmt.Errorf("requires one of fields: cn, sn, mail, userpassword")
	}

	pveuser, err := pve.client.User(context.Background(), username.ToString())
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if IsProxmoxNotFound(err) {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	err = pveuser.Update(context.Background(), proxmox.UserOptions{
		Firstname: user.CN,
		Lastname:  user.SN,
		Email:     user.Mail,
		// todo userpassword (its a separate pve endpoint)
	})
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (pve ProxmoxClient) GetUser(username common.Username) (common.User, int, error) {
	user := common.User{}
	pveuser, err := pve.client.User(context.Background(), username.ToString())
	if IsProxmoxNotFound(err) { // errors if user does not exist
		return user, http.StatusNotFound, err
	} else if err != nil {
		return user, http.StatusInternalServerError, err
	} else {
		user.Username, _ = common.ParseUsername(pveuser.UserID)
		user.CN = pveuser.Firstname
		user.SN = pveuser.Lastname
		user.Mail = pveuser.Email
		return user, http.StatusOK, nil
	}
}

func (pve ProxmoxClient) DelUser(username common.Username) (int, error) {
	user, err := pve.client.User(context.Background(), username.ToString())
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if IsProxmoxNotFound(err) {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	// assume that user cannot be nil if no error was returned
	err = user.Delete(context.Background())
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err // not authorized to delete the user
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (pve ProxmoxClient) AddUserToGroup(username common.Username, groupname common.Groupname) (int, error) {
	user, err := pve.client.User(context.Background(), username.ToString())
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if IsProxmoxNotFound(err) {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	newGroups := append(user.Groups, groupname.ToString())

	err = user.Update(context.Background(), proxmox.UserOptions{
		Groups: newGroups,
	})
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err // not authorized to delete the user
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}

func (pve ProxmoxClient) DelUserFromGroup(username common.Username, groupname common.Groupname) (int, error) {
	user, err := pve.client.User(context.Background(), username.ToString())
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err
	} else if IsProxmoxNotFound(err) {
		return http.StatusNotFound, err
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	idx := slices.Index(user.Groups, groupname.ToString())
	if idx < 0 {
		return http.StatusBadRequest, fmt.Errorf("did not find group %s in user groups {%+v}", groupname.ToString(), user.Groups)
	}
	newGroups := slices.Delete(user.Groups, idx, idx)

	err = user.Update(context.Background(), proxmox.UserOptions{
		Groups: newGroups,
	})
	if proxmox.IsNotAuthorized(err) {
		return http.StatusUnauthorized, err // not authorized to delete the user
	} else if err != nil {
		return http.StatusInternalServerError, err
	} else {
		return http.StatusOK, nil
	}
}
