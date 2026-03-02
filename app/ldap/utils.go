package ldap

import (
	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"

	common "user-manager-api/app/common"
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
			"code":    200,
			"result":  "OK",
			"message": "",
		}
	}
}
