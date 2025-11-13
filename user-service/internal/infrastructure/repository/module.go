package repository

import (
	"github.com/Nezent/microservice-template/user-service/internal/domain/user"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"repository",
	fx.Provide(
		NewUserRepository,
		fx.Annotate(
			NewUserRepository,
			fx.As(new(user.UserRepository)),
		),
	),
)
