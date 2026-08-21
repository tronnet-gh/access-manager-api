package app

type Login struct { // login body struct
	UsernameRaw string `json:"username" binding:"required"`
	Username    Username
	Password    string `json:"password" binding:"required"`
}
