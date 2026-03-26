package app

type Pool struct {
	PoolID    string         `json:"poolid"`
	Path      string         `json:"-"` // typically /pool/poolid from proxmox, only used internally
	Groups    []Group        `json:"groups"`
	Resources map[string]any `json:"resources"`
	Cluster   Cluster        `json:"cluster"`
	Templates Templates      `json:"templates"`
}

type Groupname struct { // proxmox typically formats as gid-realm for non pve realms
	GroupID string `json:"gid"`
	Realm   string `json:"realm"`
}

type Group struct {
	Groupname Groupname `json:"groupname"`
	Handler   string    `json:"-"`
	Role      string    `json:"role"`
	Users     []User    `json:"users"`
}

type Username struct { // ie userid@realm
	UserID string `json:"uid"`
	Realm  string `json:"realm"`
}

type User struct {
	Username Username `json:"username"`
	Handler  string   `json:"-"`
	CN       string   `json:"cn"` // aka first name
	SN       string   `json:"sn"` // aka last name
	Mail     string   `json:"mail"`
	Password string   `json:"password"` // only used for POST requests
}

type Cluster struct {
	Nodes map[string]bool `json:"nodes"`
	VMID  VMID            `json:"vmid"`
	//Pools   map[string]bool `json:"pools"`
	Backups Backups `json:"backups"`
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
