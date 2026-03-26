package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-ldap/ldap/v3"

	common "user-manager-api/app/common"
)

// LDAPClient wrapper struct containing the connection, baseDN, peopleDN, and groupsDN
type LDAPClient struct {
	config *common.LDAPConfig
	client *ldap.Conn
}

// returns a new LDAPClient from the config
func NewClientFromCredentials(config common.LDAPConfig, username common.Username, password string) (*LDAPClient, int, error) {
	LDAPConn, err := ldap.DialURL(config.LdapURL)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if config.StartTLS {
		err = LDAPConn.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
	}

	ldap := LDAPClient{
		config: &config,
		client: LDAPConn,
	}

	userdn := fmt.Sprintf("uid=%s,ou=people,%s", username.UserID, ldap.config.BaseDN)
	err = ldap.client.Bind(userdn, password)

	if err != nil {
		return nil, http.StatusUnauthorized, err
	} else {
		return &ldap, http.StatusOK, nil
	}
}

func (l LDAPClient) GetUser(username common.Username) (common.User, int, error) {
	user := common.User{}

	searchRequest := ldap.NewSearchRequest( //  setup search for user by uid
		fmt.Sprintf("uid=%s,ou=people,%s", username.UserID, l.config.BaseDN), // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(&(objectClass=inetOrgPerson))",                      // The filter to apply
		[]string{"dn", "cn", "sn", "mail", "uid", "memberOf"}, // A list attributes to retrieve
		nil,
	)

	searchResponse, err := l.client.Search(searchRequest) // perform search
	if err != nil {
		return user, http.StatusBadRequest, err
	}

	entry := searchResponse.Entries[0]

	user = LDAPEntryToUser(entry)

	return user, http.StatusOK, nil
}

func (l LDAPClient) NewUser(username common.Username, user common.User) (int, error) {
	if user.CN == "" || user.SN == "" || user.Password == "" || user.Mail == "" {
		return http.StatusBadRequest, ldap.NewError(
			ldap.LDAPResultUnwillingToPerform,
			errors.New("missing one of required fields: cn, sn, mail, userpassword"),
		)
	}

	addRequest := ldap.NewAddRequest(
		fmt.Sprintf("uid=%s,ou=people,%s", username.UserID, l.config.BaseDN), // DN
		nil, // controls
	)
	addRequest.Attribute("sn", []string{user.SN})
	addRequest.Attribute("cn", []string{user.CN})
	addRequest.Attribute("mail", []string{user.Mail})
	addRequest.Attribute("userPassword", []string{user.Password})
	addRequest.Attribute("objectClass", []string{"inetOrgPerson"})

	err := l.client.Add(addRequest)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func (l LDAPClient) ModUser(username common.Username, user common.User) (int, error) {
	if user.CN == "" && user.SN == "" && user.Password == "" && user.Mail == "" {
		return http.StatusBadRequest, ldap.NewError(
			ldap.LDAPResultUnwillingToPerform,
			errors.New("requires one of fields: cn, sn, mail, userpassword"),
		)
	}

	modifyRequest := ldap.NewModifyRequest(
		fmt.Sprintf("uid=%s,ou=people,%s", username.UserID, l.config.BaseDN),
		nil,
	)
	if user.CN != "" {
		modifyRequest.Replace("cn", []string{user.CN})
	}
	if user.SN != "" {
		modifyRequest.Replace("sn", []string{user.SN})
	}
	if user.Mail != "" {
		modifyRequest.Replace("mail", []string{user.Mail})
	}
	if user.Password != "" {
		modifyRequest.Replace("userPassword", []string{user.Password})
	}

	err := l.client.Modify(modifyRequest)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func (l LDAPClient) DelUser(username common.Username) (int, error) {
	userDN := fmt.Sprintf("uid=%s,ou=people,%s", username.UserID, l.config.BaseDN)

	// assumes that olcMemberOfRefint=true updates member attributes of referenced groups

	deleteUserRequest := ldap.NewDelRequest( // setup delete request
		userDN,
		nil,
	)

	err := l.client.Del(deleteUserRequest) // delete user
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func (l LDAPClient) GetGroup(groupname common.Groupname) (common.Group, int, error) {
	group := common.Group{}

	searchRequest := ldap.NewSearchRequest( //  setup search for user by uid
		fmt.Sprintf("cn=%s,ou=groups,%s", groupname.GroupID, l.config.BaseDN), // The base dn to search
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(&(objectClass=groupOfNames))", // The filter to apply
		[]string{"cn", "member"},        // A list attributes to retrieve
		nil,
	)

	searchResponse, err := l.client.Search(searchRequest) // perform search
	if err != nil {
		return group, http.StatusBadRequest, err
	}

	entry := searchResponse.Entries[0]
	group = LDAPEntryToGroup(entry)

	return group, http.StatusOK, nil
}

func (l LDAPClient) NewGroup(groupname common.Groupname) (int, error) {
	addRequest := ldap.NewAddRequest(
		fmt.Sprintf("cn=%s,ou=groups,%s", groupname.GroupID, l.config.BaseDN), // DN
		nil, // controls
	)
	addRequest.Attribute("cn", []string{groupname.GroupID})
	addRequest.Attribute("member", []string{""})
	addRequest.Attribute("objectClass", []string{"groupOfNames"})

	err := l.client.Add(addRequest)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func (l LDAPClient) ModGroup(groupname common.Groupname, group common.Group) (int, error) {
	modifyRequest := ldap.NewModifyRequest(
		fmt.Sprintf("cn=%s,ou=groups,%s", groupname.GroupID, l.config.BaseDN),
		nil,
	)

	modifyRequest.Replace("cn", []string{groupname.GroupID})

	err := l.client.Modify(modifyRequest)
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func (l LDAPClient) DelGroup(groupname common.Groupname) (int, error) {
	groupDN := fmt.Sprintf("cn=%s,ou=groups,%s", groupname.GroupID, l.config.BaseDN)

	// assumes that memberOf overlay will automatically update referenced memberOf attributes

	deleteGroupRequest := ldap.NewDelRequest( // setup delete request
		groupDN,
		nil,
	)

	err := l.client.Del(deleteGroupRequest) // delete group
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func (l LDAPClient) AddUserToGroup(username common.Username, groupname common.Groupname) (int, error) {
	userDN := fmt.Sprintf("uid=%s,ou=people,%s", username.UserID, l.config.BaseDN)
	groupDN := fmt.Sprintf("cn=%s,ou=groups,%s", groupname.GroupID, l.config.BaseDN)

	modifyRequest := ldap.NewModifyRequest( // modify group member value
		groupDN,
		nil,
	)

	modifyRequest.Add("member", []string{userDN}) // add user to group member attribute

	err := l.client.Modify(modifyRequest) // modify group
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}

func (l LDAPClient) DelUserFromGroup(username common.Username, groupname common.Groupname) (int, error) {
	userDN := fmt.Sprintf("uid=%s,ou=people,%s", username.UserID, l.config.BaseDN)
	groupDN := fmt.Sprintf("cn=%s,ou=groups,%s", groupname.GroupID, l.config.BaseDN)

	modifyRequest := ldap.NewModifyRequest( // modify group member value
		groupDN,
		nil,
	)

	modifyRequest.Delete("member", []string{userDN}) // remove user from group member attribute

	err := l.client.Modify(modifyRequest) // modify group
	if err != nil {
		return http.StatusBadRequest, err
	}

	return http.StatusOK, nil
}
