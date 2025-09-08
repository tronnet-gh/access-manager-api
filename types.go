package main

type User struct {
	DN        string         `json:"dn"`
	UID       string         `json:"uid"`
	CN        string         `json:"cn"`
	SN        string         `json:"sn"`
	Mail      string         `json:"mail"`
	MemberOf  []string       `json:"memberOf"`
	Resources map[string]any `json:"resources"`
	Cluster   Cluster        `json:"cluster"`
	Templates Templates      `json:"templates"`
}

type Cluster struct {
	Admin   bool            `json:"admin"`
	Nodes   map[string]bool `json:"nodes"`
	VMID    VMID            `json:"vmid"`
	Pools   map[string]bool `json:"pools"`
	Backups Backups         `json:"backups"`
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
