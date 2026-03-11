package app

import (
	"fmt"
	"strings"
)

// returns an error if the groupname format was not correct
func ParseGroupname(groupname string) (Groupname, error) {
	g := Groupname{}
	x := strings.Split(groupname, "-")
	if len(x) == 1 {
		g.GroupID = groupname
		g.Realm = "pve"
		return g, nil
	} else if len(x) == 2 {
		g.GroupID = x[0]
		g.Realm = x[1]
		return g, nil
	} else {
		return g, fmt.Errorf("groupid did not follow the format <groupid> or <groupid>-<realm>")
	}
}

// returns an error if the username format was not correct
func ParseUsername(username string) (Username, error) {
	u := Username{}
	x := strings.Split(username, "@")
	if len(x) == 2 {
		u.UserID = x[0]
		u.Realm = x[1]
		return u, nil
	} else {
		return u, fmt.Errorf("userid did not follow the format <userid>@<realm>")
	}
}

func (g Groupname) ToString() string {
	return fmt.Sprintf("%s-%s", g.GroupID, g.Realm)
}

func (u Username) ToString() string {
	return fmt.Sprintf("%s-%s", u.UserID, u.Realm)
}
