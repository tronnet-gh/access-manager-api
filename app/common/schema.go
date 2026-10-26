package app

type Login struct { // login body struct
	UsernameRaw string `json:"username"`
	Username    Username
	Password    string `json:"password"`
}
