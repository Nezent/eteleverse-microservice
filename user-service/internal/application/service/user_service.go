package service

import (
	"github.com/Nezent/microservice-template/user-service/internal/application/dto"
	"github.com/Nezent/microservice-template/user-service/internal/domain/shared"
	"github.com/Nezent/microservice-template/user-service/internal/domain/user"
)

type UserServiceImpl struct {
	repo user.UserRepository
}

func NewUserService(repo user.UserRepository) *UserServiceImpl {
	return &UserServiceImpl{
		repo: repo,
	}
}

// Compile-time interface check
var _ user.UserService = (*UserServiceImpl)(nil)

// Implement service methods here
func (s *UserServiceImpl) CreateUser(req *dto.CreateUserRequest) (*dto.CreateUserResponse, *shared.DomainError) {
	// Implementation goes here
	user := &user.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password, // In real scenarios, ensure to hash the password
	}
	id, err := s.repo.CreateUser(user)
	if err != nil {
		return nil, err
	}
	return &dto.CreateUserResponse{ID: id.String()}, nil
}

func (s *UserServiceImpl) GetUser() (*dto.GetUserResponse, *shared.DomainError) {
	// Implementation goes here
	users, err := s.repo.GetUser()
	if err != nil {
		return nil, err
	}
	if users == nil || len(*users) == 0 {
		return &dto.GetUserResponse{Users: []dto.UserDetail{}}, nil
	}

	userDetails := make([]dto.UserDetail, len(*users))
	for i, u := range *users {
		userDetails[i] = dto.UserDetail{
			ID:    u.ID.String(),
			Name:  u.Name,
			Email: u.Email,
		}
	}
	return &dto.GetUserResponse{Users: userDetails}, nil
}
