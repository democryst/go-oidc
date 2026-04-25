package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openbao/openbao/api/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/democryst/go-oidc/internal/api/handlers"
	"github.com/democryst/go-oidc/internal/api/middleware"
	"github.com/democryst/go-oidc/internal/config"
	"github.com/democryst/go-oidc/internal/core/oidc"
	"github.com/democryst/go-oidc/internal/crypto/cipher"
	"github.com/democryst/go-oidc/internal/crypto/hashing"
	"github.com/democryst/go-oidc/internal/crypto/signer"
	"github.com/democryst/go-oidc/internal/repository/postgres"
)

func main() {
	// 1. Config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	// 2. Database
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	pgCfg, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("parse db dsn: %v", err)
	}
	pgCfg.MaxConns = int32(cfg.Database.MaxConns)

	pool, err := pgxpool.NewWithConfig(dbCtx, pgCfg)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	repo := postgres.NewPostgresRepository(pool)
	// Phase 2: Wrap repo with BatchRepository for high-load audit logging
	batchRepo := postgres.NewBatchRepository(repo, pool, 1000, 500*time.Millisecond)
	defer batchRepo.Close()

	// 3. OpenBao (KMS)
	baoCfg := api.DefaultConfig()
	baoCfg.Address = cfg.OpenBao.Address
	baoClient, err := api.NewClient(baoCfg)
	if err != nil {
		log.Fatalf("openbao client: %v", err)
	}
	baoClient.SetToken(cfg.OpenBao.Token)

	// 4. Crypto Layers
	hasher := hashing.NewSHA3Hasher()
	encryptor := cipher.NewTransitEncryptor(baoClient, cfg.OpenBao.TransitMount, cfg.OpenBao.EncryptionKeyName)
	
	// Classical Signer (Ed25519 via OpenBao)
	classicalSigner := signer.NewOpenBaoSigner(baoClient, cfg.OpenBao.TransitMount, cfg.OpenBao.Ed25519KeyName)

	// Dual Signer (Nested JWS: Ed25519 + Dilithium3)
	// For production, the DilithiumKeyID would be configured or looked up.
	// Here we use a dummy or looked up key from repo.
	pqcKeyFetcher := signer.CreatePQCFetcher(repo, encryptor, [16]byte{}) // Placeholder key ID
	dualSigner := signer.NewDualSigner(classicalSigner, pqcKeyFetcher)

	// 5. OIDC Service
	svc := oidc.NewOIDCService(batchRepo, dualSigner, hasher, cfg)

	// 6. Routing & Middleware
	var rlStore middleware.RateLimitStore
	if cfg.Redis.Address != "" {
		rdb := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Address,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		// Check connection
		rCtx, rCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := rdb.Ping(rCtx).Err(); err != nil {
			log.Printf("Warning: Redis connect failed (%v), falling back to MemoryStore", err)
			rlStore = middleware.NewMemoryStore()
		} else {
			log.Printf("Connected to Redis at %s for distributed rate limiting", cfg.Redis.Address)
			rlStore = middleware.NewRedisStore(rdb)
		}
		rCancel()
	} else {
		rlStore = middleware.NewMemoryStore()
	}

	h := handlers.NewOIDCHandler(svc)
	rl := middleware.RateLimit(rlStore, cfg.Server.RateLimit, time.Minute)

	// --- Admin Dashboard (Phase 3) ---
	adminHandler := handlers.NewAdminHandler(svc)
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/admin/stats", adminHandler.HandleStats)
	adminMux.HandleFunc("/admin/clients", adminHandler.HandleClients)
	adminMux.HandleFunc("/admin/clients/create", adminHandler.HandleCreateClient)
	adminMux.HandleFunc("/admin/rotate-keys", adminHandler.HandleRotateKeys)
	adminMux.HandleFunc("/admin/audit", adminHandler.HandleAuditLogs)

	// Protected Admin API
	protectedAdmin := middleware.AdminAuth(adminMux)

	mux := http.NewServeMux()
	// Public OIDC Endpoints
	mux.Handle("/authorize", rl(http.HandlerFunc(h.HandleAuthorize)))
	mux.Handle("/token", rl(http.HandlerFunc(h.HandleToken)))
	mux.HandleFunc("/.well-known/openid-configuration", h.HandleDiscovery)
	mux.HandleFunc("/.well-known/jwks.json", h.HandleJWKS)

	// Admin API
	mux.Handle("/admin/", protectedAdmin)

	// Health Probes
	healthHandler := handlers.NewHealthHandler()
	mux.HandleFunc("/live", healthHandler.Liveness)
	mux.HandleFunc("/ready", healthHandler.Readiness)

	// Static UI
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/admin" || r.URL.Path == "/admin/" {
			http.ServeFile(w, r, "web/admin/index.html")
			return
		}
		http.NotFound(w, r)
	})

	// Prometheus Metrics
	mux.Handle("/metrics", promhttp.Handler())

	// Apply Metrics/Logger to all
	handler := middleware.Metrics(middleware.Logger(mux))

	// 7. Server
	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("OIDC Provider starting on %s (Issuer: %s)", cfg.Server.Addr, cfg.OIDC.Issuer)
		if err := srv.ListenAndServeTLS(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// 8. Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("Exited")
}
