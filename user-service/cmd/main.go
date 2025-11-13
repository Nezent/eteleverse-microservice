package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Nezent/microservice-template/user-service/config"
	"github.com/Nezent/microservice-template/user-service/internal/application/service"
	"github.com/Nezent/microservice-template/user-service/internal/infrastructure/database"
	"github.com/Nezent/microservice-template/user-service/internal/infrastructure/logger"
	"github.com/Nezent/microservice-template/user-service/internal/infrastructure/repository"
	"github.com/Nezent/microservice-template/user-service/internal/interface/handler"
	"github.com/Nezent/microservice-template/user-service/internal/interface/routes"
	"github.com/Nezent/microservice-template/user-service/pkg/router"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		config.Module,
		router.Module,
		routes.Module,
		database.Module,
		handler.Module,
		service.Module,
		repository.Module,
		logger.Module,
		fx.Invoke(func(
			router *chi.Mux,
			routes *routes.APIV1Routes,
			lc fx.Lifecycle,
		) {
			lc.Append(fx.Hook{
				OnStart: func(_ context.Context) error {
					// Register routes
					routes.Register()
					log.Printf("Server started on port %v", 8080)
					go func() {
						if err := http.ListenAndServe(fmt.Sprintf(":%d", 8080), router); err != nil {
							log.Fatalf("failed to start server: %v", err)
						}
					}()
					return nil
				},
				OnStop: func(_ context.Context) error {
					return nil
				},
			})
		}),
	)
	app.Run()
}
