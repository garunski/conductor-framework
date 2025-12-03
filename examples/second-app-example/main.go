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
	cfg.AppName = "SecondApp"
	cfg.AppVersion = getVersion()
	cfg.ManifestFS = manifestFS
	cfg.ManifestRoot = "manifests"
	
	// Configure a different CRD for this app
	// This will use: appparameters.mycompany.io
	cfg.CRDGroup = "mycompany.io"
	cfg.CRDVersion = "v1alpha1"
	cfg.CRDResource = "appparameters"

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

