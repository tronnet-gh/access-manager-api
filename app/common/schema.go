package app

type Login struct { // login body struct
	UsernameRaw string `form:"username" binding:"required"`
	Username    Username
	Password    string `form:"password" binding:"required"`
}
