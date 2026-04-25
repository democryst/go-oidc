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
	svc := oidc.NewOIDCService(repo, dualSigner, hasher, cfg)

	// 6. Routing & Middleware
	h := handlers.NewOIDCHandler(svc)
	rlStore := middleware.NewMemoryStore()
	rl := middleware.RateLimit(rlStore, cfg.Server.RateLimit, time.Minute)

	mux := http.NewServeMux()
	mux.Handle("/authorize", rl(http.HandlerFunc(h.HandleAuthorize)))
	mux.Handle("/token", rl(http.HandlerFunc(h.HandleToken)))
	mux.HandleFunc("/.well-known/openid-configuration", h.HandleDiscovery)
	mux.HandleFunc("/.well-known/jwks.json", h.HandleJWKS)

	// Apply Logger to all
	handler := middleware.Logger(mux)

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
