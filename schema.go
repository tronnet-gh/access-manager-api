package main

type User struct {
	Id        string
	Password  string `form:"userpassword" binding:"required"`
	CN        string `form:"usercn" binding:"required"`
	SN        string `form:"usersn" binding:"required"`
	Resources struct{}
	Cluster   struct{}
	Templates struct{}
}
