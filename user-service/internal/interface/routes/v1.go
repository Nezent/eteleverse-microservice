package routes

import (
	"github.com/Nezent/microservice-template/user-service/internal/interface/handler"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

type APIV1RoutesParams struct {
	fx.In

	Router      *chi.Mux
	UserHandler *handler.UserHandler
}

type APIV1Routes struct {
	router      *chi.Mux
	userHandler *handler.UserHandler
}

func NewRoutes(params APIV1RoutesParams) *APIV1Routes {
	return &APIV1Routes{
		router:      params.Router,
		userHandler: params.UserHandler,
	}
}

func (r *APIV1Routes) Register() {
	r.router.Route("/api/v1", func(v1 chi.Router) {
		// guest routes
		v1.Route("/auth", func(noAuth chi.Router) {
			// noAuth.Post("/login", r.userHandler.Login)
			noAuth.Post("/register", r.userHandler.Register)
		})
	})
}
