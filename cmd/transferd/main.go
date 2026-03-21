package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/luxfi/transfer/pkg/comms"
	"github.com/luxfi/transfer/pkg/disclosures"
	"github.com/luxfi/transfer/pkg/dividends"
	"github.com/luxfi/transfer/pkg/filings"
	"github.com/luxfi/transfer/pkg/ledger"
	"github.com/luxfi/transfer/pkg/restrictions"
	"github.com/luxfi/transfer/pkg/router"
	"github.com/luxfi/transfer/pkg/shareholder"
	"github.com/luxfi/transfer/pkg/store"
	"github.com/luxfi/transfer/pkg/voting"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	listenAddr := envOr("TRANSFER_LISTEN", ":8092")

	st := store.NewMemoryStore()

	shareholderSvc := shareholder.New(st)
	ledgerSvc := ledger.New(st)
	restrictionsSvc := restrictions.New(st)
	disclosuresSvc := disclosures.New(st)
	commsSvc := comms.New(st)
	dividendsSvc := dividends.New(st)
	filingsSvc := filings.New(st)
	votingSvc := voting.New(st)

	r := router.New(
		shareholderSvc,
		ledgerSvc,
		restrictionsSvc,
		disclosuresSvc,
		commsSvc,
		dividendsSvc,
		filingsSvc,
		votingSvc,
	)

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		log.Info().Str("addr", listenAddr).Msg("Transfer Agent API starting")
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info().Msg("Shutting down...")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutCancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Error().Err(err).Msg("Shutdown error")
		}
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
