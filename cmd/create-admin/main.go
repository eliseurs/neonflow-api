package main

import (
	"context"
	"fmt"
	"os"

	"neonflow-api/internal/config"
	"neonflow-api/internal/domain/authentication"
	"neonflow-api/internal/domain/authentication/repository"
	"neonflow-api/internal/domain/authentication/service"
	"neonflow-api/internal/infra/db"
	"neonflow-api/internal/pkg/logger"
	"neonflow-api/internal/pkg/security"
)

// create-admin cria um usuário da central. Lê os dados de variáveis de ambiente
// para a senha não aparecer na lista de processos nem no histórico do shell.
//
//	ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=... [ADMIN_ROLE=admin|central] ./create-admin
func main() {
	cfg := config.Load()
	log := logger.New(cfg.IsProduction())
	ctx := context.Background()

	database, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	defer database.Close() //nolint:errcheck

	role := os.Getenv("ADMIN_ROLE")
	if role == "" {
		role = authentication.RoleAdmin
	}

	svc := service.NewAuthService(repository.NewAuthRepository(database), security.NewBcryptHasher(), nil, log)
	user, err := svc.CreateUser(ctx, authentication.CreateUserRequest{
		Name:     os.Getenv("ADMIN_NAME"),
		Email:    os.Getenv("ADMIN_EMAIL"),
		Password: os.Getenv("ADMIN_PASSWORD"),
		Role:     role,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
	fmt.Printf("usuário criado: %s (%s, %s)\n", user.Email, user.Role, user.ID)
}
