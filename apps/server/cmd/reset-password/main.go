package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/yuudi/cloudmusic/apps/server/internal/auth"
	"github.com/yuudi/cloudmusic/apps/server/internal/config"
	"github.com/yuudi/cloudmusic/apps/server/internal/database"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: reset-password USERNAME (password is read from terminal)")
	}
	if !term.IsTerminal(int(syscall.Stdin)) {
		return fmt.Errorf("a terminal is required")
	}
	fmt.Fprint(os.Stderr, "New password: ")
	p, e := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if e != nil {
		return e
	}
	defer clear(p)
	fmt.Fprint(os.Stderr, "Confirm password: ")
	confirmation, e := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if e != nil {
		return e
	}
	defer clear(confirmation)
	if string(p) != string(confirmation) {
		return fmt.Errorf("passwords do not match")
	}
	cfg, e := config.LoadDatabase()
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, e := database.Open(ctx, cfg)
	if e != nil {
		return e
	}
	defer pool.Close()
	e = (&auth.Service{DB: pool}).ResetPassword(ctx, strings.TrimSpace(os.Args[1]), string(p))
	if e != nil {
		return fmt.Errorf("password reset failed: %w", e)
	}
	fmt.Fprintln(os.Stderr, "Password reset; all sessions revoked.")
	return nil
}
