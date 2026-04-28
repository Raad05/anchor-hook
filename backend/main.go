// anchor-hook – Stage 2: WebSocket Listener & Log Decoder
//
// Reads two environment variables:
//   PROGRAM_ID   – Solana program address to monitor (default: DMSM65cnaykxbmPLdaam9QFeJ1CuuEDbMpibNHm45ZbD)
//   RPC_WS_URL   – WebSocket URL of the Solana RPC node (default: ws://127.0.0.1:8900)
//
// The process connects, subscribes to program logs, decodes any UserAction
// events it finds, and prints them to stdout.  The channel-based handoff is
// designed so that Stage 3's dispatcher can be wired in by reading the same
// channel instead of (or in addition to) printing.
package main

import (
	"log"
	"os"

	"github.com/Raad05/anchor-hook/decoder"
	"github.com/Raad05/anchor-hook/listener"
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

	log.Printf("[main] starting anchor-hook listener")
	log.Printf("[main] program  = %s", programID)
	log.Printf("[main] ws URL   = %s", wsURL)

	// ── Connect ──────────────────────────────────────────────────────────────
	l, err := listener.Connect(wsURL)
	if err != nil {
		log.Fatalf("[main] connect: %v", err)
	}
	defer l.Close()

	// ── Subscribe ─────────────────────────────────────────────────────────────
	if err := l.Subscribe(programID); err != nil {
		log.Fatalf("[main] subscribe: %v", err)
	}

	// ── Listen ────────────────────────────────────────────────────────────────
	// ch is the Stage 3 handoff point: replace the print loop below with a
	// dispatcher.Dispatch(ch) call when Stage 3 is implemented.
	ch := make(chan listener.RawLog, 64)
	go l.Listen(ch)

	log.Printf("[main] listening for events… (Ctrl-C to stop)")

	for rawLog := range ch {
		b64, ok := decoder.FindProgramData(rawLog.Logs)
		if !ok {
			// Transaction from our program but no event data (shouldn't happen
			// with trigger_action, but guard anyway).
			continue
		}

		ua, err := decoder.DecodeUserAction(b64)
		if err != nil {
			log.Printf("[decoder] skip tx %s: %v", rawLog.Signature[:8], err)
			continue
		}

		log.Printf("[decoded] sig=%s… user=%s action=%s amount=%d",
			rawLog.Signature[:8],
			ua.UserBase58(),
			ua.ActionType,
			ua.Amount,
		)
	}

	log.Printf("[main] channel closed, exiting")
}
