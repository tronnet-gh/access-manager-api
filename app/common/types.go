package app

type Backend interface {
	NewPool(poolname string) (int, error)
	DelPool(poolname string) (int, error)
	NewGroup(groupname Groupname) (int, error)
	DelGroup(groupname Groupname) (int, error)
	AddGroupToPool(groupname Groupname, poolname string) (int, error)
	DelGroupFromPool(groupname Groupname, poolname string) (int, error)
	NewUser(username Username, user User) (int, error)
	DelUser(username Username) (int, error)
	AddUserToGroup(username Username, groupname Groupname) (int, error)
	DelUserFromGroup(username Username, groupname Groupname) (int, error)
}

type Pool struct {
	PoolID       string          `json:"poolid"`
	Path         string          `json:"-"` // typically /pool/poolid from proxmox, only used internally
	Groups       []Group         `json:"groups"`
	Resources    map[string]any  `json:"resources"`
	Templates    Templates       `json:"templates"`
	AllowedNodes map[string]bool `json:"nodes-allowed"`
	VMIDRange    VMID            `json:"vmid-allowed"`
	Backups      Backups         `json:"backups-allowed"` // measured in numbers
}

// proxmox typically formats as gid-realm for non pve realms
// proxmox realms are formatted without realm values
// I assume that backends store groups by ID only and only proxmox will append the realm string
type Groupname struct {
	GroupID string `json:"gid"`
	Realm   string `json:"realm"`
}

type Group struct {
	Groupname Groupname `json:"groupname"`
	Role      string    `json:"role"` // role in owner pool
	Users     []User    `json:"users"`
}

type Username struct { // ie userid@realm
	UserID string `json:"uid"`
	Realm  string `json:"realm"`
}

type User struct {
	Username Username `json:"username"`
	CN       string   `json:"cn"` // aka first name
	SN       string   `json:"sn"` // aka last name
	Mail     string   `json:"mail"`
	Password string   `json:"password"` // only used for POST requests
}

type VMID struct {
	Min int `json:"min"`
	MAx int `json:"max"`
}

type Backups struct {
	Max int `json:"max"`
}

type Templates struct {
	Instances struct {
		LXC  map[string]ResourceTemplate `json:"lxc"`
		QEMU map[string]ResourceTemplate `json:"qemu"`
	} `json:"instances"`
}

type SimpleResource struct {
	Limits struct {
		Global SimpleLimit            `json:"global"`
		Nodes  map[string]SimpleLimit `json:"nodes"`
	} `json:"limits"`
}

type SimpleLimit struct {
	Max int `json:"max"`
}

type MatchResource struct {
	Limits struct {
		Global []MatchLimit            `json:"global"`
		Nodes  map[string][]MatchLimit `json:"nodes"`
	} `json:"limits"`
}

type MatchLimit struct {
	Match string `json:"match"`
	Name  string `json:"name"`
	Max   int    `json:"max"`
}

type ResourceTemplate struct {
	Value    string `json:"value"`
	Resource struct {
		Enabled bool   `json:"enabled"`
		Name    string `json:"name"`
		Amount  int    `json:"amount"`
	} `json:"resource"`
}
