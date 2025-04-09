package inmemory

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	status "github.com/gogo/status"
	contracts_users "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/users"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"
	zerolog "github.com/rs/zerolog"
	codes "google.golang.org/grpc/codes"
)

type (
	service struct {
		config   *contracts_users.UserStoreConfig
		users    []*contracts_users.User
		mapUsers map[string]*contracts_users.User
	}
)

var stemService = (*service)(nil)
var _ contracts_users.IUserStore = (*service)(nil)

func (s *service) Ctor(config *contracts_users.UserStoreConfig) (contracts_users.IUserStore, error) {

	var users []*contracts_users.User

	usersData, err := LoadUsersData(config.UserFile)
	if err != nil {
		return nil, err
	}
	mapUsers := make(map[string]*contracts_users.User)
	for idx := range usersData.Users {
		usersData.Users[idx].Username = strings.ToLower(usersData.Users[idx].Username)
		for aid := range usersData.Users[idx].AllowedAccounts {
			usersData.Users[idx].AllowedAccounts[aid] = strings.ToLower(usersData.Users[idx].AllowedAccounts[aid])
		}
		user := usersData.Users[idx]
		users = append(users, &user)
		mapUsers[user.Username] = &user
	}
	return &service{
		config:   config,
		users:    users,
		mapUsers: mapUsers,
	}, nil
}
func AddSingletonUserStore(builder di.ContainerBuilder) {
	di.AddSingleton[contracts_users.IUserStore](
		builder,
		stemService.Ctor,
	)
}
func LoadUsersData(filename string) (*contracts_users.Users, error) {
	if !shared.FileExists(filename) {
		return nil, fmt.Errorf("file does not exist: %s", filename)
	}
	data := shared.LoadFile(filename)
	usersData := &contracts_users.Users{}
	err := json.Unmarshal([]byte(data), usersData)
	return usersData, err
}

func (s *service) GetUsers(ctx context.Context) (*contracts_users.GetUsersResponse, error) {
	return &contracts_users.GetUsersResponse{
		Users: s.users,
	}, nil
}
func (s *service) validateGetUserByUserNameRequest(request *contracts_users.GetUserByUserNameRequest) error {
	if fluffycore_utils.IsNil(request) {
		return status.Error(codes.InvalidArgument, "request is nil")
	}
	if fluffycore_utils.IsEmptyOrNil(request.UserName) {
		return status.Error(codes.InvalidArgument, "UserName is empty")
	}
	request.UserName = strings.ToLower(request.UserName)
	return nil
}
func (s *service) GetUserByUserName(ctx context.Context, request *contracts_users.GetUserByUserNameRequest) (*contracts_users.GetUserByUserNameResponse, error) {
	if err := s.validateGetUserByUserNameRequest(request); err != nil {
		return nil, err
	}
	log := zerolog.Ctx(ctx).With().Str("command", "GetUserByUserName").Logger()
	log = log.With().Str("userName", request.UserName).Logger()
	user, ok := s.mapUsers[request.UserName]
	if !ok {
		log.Debug().Msg("user not found")
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return &contracts_users.GetUserByUserNameResponse{
		User: user,
	}, nil
}
func (s *service) validateAuthenticateUserRequest(request *contracts_users.AuthenticateUserRequest) error {
	if fluffycore_utils.IsNil(request) {
		return status.Error(codes.InvalidArgument, "request is nil")
	}
	if fluffycore_utils.IsEmptyOrNil(request.UserName) {
		return status.Error(codes.InvalidArgument, "UserName is empty")
	}
	if fluffycore_utils.IsEmptyOrNil(request.Password) {
		return status.Error(codes.InvalidArgument, "Password is empty")
	}
	if fluffycore_utils.IsEmptyOrNil(request.Account) {
		return status.Error(codes.InvalidArgument, "Account is empty")
	}
	request.Account = strings.ToLower(request.Account)
	return nil
}

func (s *service) AuthenticateUser(ctx context.Context, request *contracts_users.AuthenticateUserRequest) (*contracts_users.AuthenticateUserResponse, error) {
	if err := s.validateAuthenticateUserRequest(request); err != nil {
		return nil, err
	}
	getUserByUserNameResponse, err := s.GetUserByUserName(ctx, &contracts_users.GetUserByUserNameRequest{
		UserName: request.UserName,
	})
	if err != nil {
		return nil, err
	}
	user := getUserByUserNameResponse.User
	if user.Password != request.Password {
		return nil, status.Error(codes.Unauthenticated, "invalid password")
	}
	if slices.Contains(user.AllowedAccounts, request.Account) {
		return &contracts_users.AuthenticateUserResponse{
			User: user,
		}, nil
	}
	if slices.Contains(user.AllowedAccounts, "*") {
		return &contracts_users.AuthenticateUserResponse{
			User: user,
		}, nil
	}
	return nil, status.Error(codes.Unauthenticated, "user not allowed to access this account")
}
