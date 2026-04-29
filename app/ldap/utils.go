package ldap

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"

	common "access-manager-api/app/common"
)

func LDAPEntryToUser(entry *ldap.Entry) common.User {
	return common.User{
		CN:   entry.GetAttributeValue("cn"),
		SN:   entry.GetAttributeValue("sn"),
		Mail: entry.GetAttributeValue("mail"),
	}
}

func LDAPEntryToGroup(entry *ldap.Entry) common.Group {
	return common.Group{}
}

func ParseLDAPError(err error) gin.H {
	if err != nil {
		LDAPerr := err.(*ldap.Error)
		return gin.H{
			"ok":      false,
			"code":    LDAPerr.ResultCode,
			"result":  ldap.LDAPResultCodeMap[LDAPerr.ResultCode],
			"message": LDAPerr.Err.Error(),
		}
	} else {
		return gin.H{
			"ok":      true,
			"code":    http.StatusOK,
			"result":  "OK",
			"message": "",
		}
	}
}

func ExtractUIDFromUserDN(dn string) (string, error) {
	m := regexp.MustCompilePOSIX("uid=([[:alnum:]]+)")
	x := m.FindStringSubmatch(dn)
	if len(x) != 2 {
		return "", fmt.Errorf("could not find uid in dn %s", dn)
	}
	return x[1], nil
}
