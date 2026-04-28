// anchor-hook – Stage 3: Webhook Registration & Dispatcher
//
// Environment variables:
//
//	PROGRAM_ID   – Solana program address to monitor (default: DMSM65cnaykxbmPLdaam9QFeJ1CuuEDbMpibNHm45ZbD)
//	RPC_WS_URL   – WebSocket URL of the Solana RPC node (default: ws://127.0.0.1:8900)
//	API_PORT     – HTTP port for the REST API (default: 8080)
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Raad05/anchor-hook/api"
	"github.com/Raad05/anchor-hook/decoder"
	"github.com/Raad05/anchor-hook/dispatcher"
	"github.com/Raad05/anchor-hook/listener"
	"github.com/Raad05/anchor-hook/registry"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	programID := getEnv("PROGRAM_ID", "DMSM65cnaykxbmPLdaam9QFeJ1CuuEDbMpibNHm45ZbD")
	wsURL := getEnv("RPC_WS_URL", "ws://127.0.0.1:8900")
	apiAddr := ":" + getEnv("API_PORT", "8080")

	log.Printf("[main] starting anchor-hook")
	log.Printf("[main] program  = %s", programID)
	log.Printf("[main] ws URL   = %s", wsURL)
	log.Printf("[main] api addr = %s", apiAddr)

	// ── Shared state ──────────────────────────────────────────────────────────
	reg := registry.New()

	// ── Dispatcher ────────────────────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	disp := dispatcher.New(reg)
	disp.Start(ctx)

	// ── REST API ──────────────────────────────────────────────────────────────
	apiServer := api.New(reg, apiAddr)
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Fatalf("[api] server error: %v", err)
		}
	}()

	// ── WebSocket Listener ────────────────────────────────────────────────────
	l, err := listener.Connect(wsURL)
	if err != nil {
		log.Fatalf("[main] connect: %v", err)
	}
	defer l.Close()

	if err := l.Subscribe(programID); err != nil {
		log.Fatalf("[main] subscribe: %v", err)
	}

	ch := make(chan listener.RawLog, 64)
	go l.Listen(ch)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("[main] ready — POST http://localhost%s/register-webhook to add a webhook", apiAddr)

	for {
		select {
		case <-sig:
			log.Printf("[main] shutting down")
			cancel()
			return

		case rawLog, ok := <-ch:
			if !ok {
				log.Printf("[main] listener channel closed, exiting")
				return
			}

			b64, found := decoder.FindProgramData(rawLog.Logs)
			if !found {
				continue
			}

			ua, err := decoder.DecodeUserAction(b64)
			if err != nil {
				log.Printf("[main] decode error (sig=%s…): %v", rawLog.Signature[:8], err)
				continue
			}

			log.Printf("[main] event: user=%s action=%s amount=%d",
				ua.UserBase58(), ua.ActionType, ua.Amount)

			disp.Dispatch(ua)
		}
	}
}
