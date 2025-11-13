package repository

import (
	"context"
	"time"

	"github.com/Nezent/microservice-template/user-service/internal/domain/shared"
	"github.com/Nezent/microservice-template/user-service/internal/domain/user"
	"github.com/Nezent/microservice-template/user-service/internal/infrastructure/database"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UserRepositoryImpl struct {
	db *database.Database
}

// Compile-time interface check
var _ user.UserRepository = (*UserRepositoryImpl)(nil)

func NewUserRepository(db *database.Database) *UserRepositoryImpl {
	return &UserRepositoryImpl{
		db: db,
	}
}

func (r *UserRepositoryImpl) CreateUser(user *user.User) (uuid.UUID, *shared.DomainError) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start transaction
	err := r.db.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Insert user and return the generated ID
		_, err := tx.NewInsert().Model(user).Returning("id").Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, shared.NewDomainError("CREATE_FAILED", 500, err.Error())
	}
	return user.ID, nil
}

func (r *UserRepositoryImpl) GetUser() (*[]user.User, *shared.DomainError) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var users []user.User
	err := r.db.DB.NewSelect().Model(&users).Scan(ctx)
	if err != nil {
		return nil, shared.NewDomainError("FETCH_FAILED", 500, err.Error())
	}
	return &users, nil
}
