package app

type Login struct { // login body struct
	UsernameRaw string `form:"username" binding:"required"`
	Username    Username
	Password    string `form:"password" binding:"required"`
}

type UserFormRequired struct { // add user body struct
	CN       string `form:"cn" binding:"required"`
	SN       string `form:"sn" binding:"required"`
	Mail     string `form:"mail" binding:"required"`
	Password string `form:"password" binding:"required"`
}

type UserFormOptional struct { // modify user body struct
	CN       string `form:"cn"`
	SN       string `form:"sn"`
	Mail     string `form:"mail"`
	Password string `form:"password"`
}
