package main

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/garunski/conductor-framework/pkg/framework"
)

//go:embed manifests
var manifestFS embed.FS

func main() {
	ctx := context.Background()

	cfg := framework.DefaultConfig()
	cfg.AppName = "Guestbook"
	cfg.AppVersion = getVersion()
	cfg.ManifestFS = manifestFS
	cfg.ManifestRoot = "manifests"

	if err := framework.Run(ctx, cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func getVersion() string {
	if v := os.Getenv("VERSION"); v != "" {
		return v
	}
	return "dev"
}

