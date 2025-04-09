package users

import "context"

type (
	UserStoreConfig struct {
		UserFile string `json:"userFile"`
	}
	Permissions struct {
		Allow []string `json:"allow"`
		Deny  []string `json:"deny"`
	}
	User struct {
		Username        string      `json:"username"`
		Password        string      `json:"password"`
		Sub             Permissions `json:"sub"`
		Pub             Permissions `json:"pub"`
		AllowedAccounts []string    `json:"allowedAccounts"`
	}
	Users struct {
		Users []User `json:"users"`
	}
	GetUserByUserNameRequest struct {
		UserName string `json:"userName"`
	}
	GetUserByUserNameResponse struct {
		User *User `json:"user"`
	}
	GetUsersResponse struct {
		Users []*User `json:"users"`
	}
	AuthenticateUserRequest struct {
		UserName string `json:"userName"`
		Password string `json:"password"`
		Account  string `json:"account"`
	}
	AuthenticateUserResponse struct {
		User *User `json:"user"`
	}
	IUserStore interface {
		GetUsers(ctx context.Context) (*GetUsersResponse, error)
		AuthenticateUser(ctx context.Context, request *AuthenticateUserRequest) (*AuthenticateUserResponse, error)
		GetUserByUserName(ctx context.Context, request *GetUserByUserNameRequest) (*GetUserByUserNameResponse, error)
	}
)
