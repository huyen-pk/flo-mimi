package main

import (
	"log"
	"net/http"
	"os"
	"time"

	contentadapter "platform/internal/adapters/content"
	analyticsadapter "platform/internal/adapters/analytics"
	gatewayadapter "platform/internal/adapters/gateway"
	bootstrapapp "platform/internal/application/bootstrap"
	interactionapp "platform/internal/application/interactions"
	httpapp "platform/internal/http"
	webassets "platform/web"
)

func main() {
	bootstrapRepository, err := contentadapter.NewAppDBRepository(
		envOrDefault("APPDB_DSN", "postgresql://analytics:analytics@localhost:5432/analytics?sslmode=disable"),
		envOrDefault("PLATFORM_ENVIRONMENT", "local"),
	)
	if err != nil {
		log.Fatalf("create bootstrap repository: %v", err)
	}
	defer bootstrapRepository.Close()

	httpClient := &http.Client{Timeout: 8 * time.Second}
	eventGateway := gatewayadapter.NewHTTPEventGateway(envOrDefault("EVENT_GATEWAY_BASE_URL", "http://localhost:8000"), httpClient)

	analyticsRepo, err := analyticsadapter.NewClickHouseRepository(envOrDefault("CLICKHOUSE_HOST", "clickhouse"), envOrDefault("CLICKHOUSE_PORT", "8123"))
	if err != nil {
		log.Fatalf("create analytics repository: %v", err)
	}

	bootstrapService := bootstrapapp.NewService(bootstrapRepository, analyticsRepo)
	interactionService := interactionapp.NewService(eventGateway, bootstrapRepository)
	handler := httpapp.NewHandler(bootstrapService, interactionService, webassets.Dist())

	server := &http.Server{
		Addr:              ":" + envOrDefault("PORT", "8081"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("platform app listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve http: %v", err)
	}
}

func envOrDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
