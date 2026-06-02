package config

// Version is set via ldflags at build time:
//   go build -ldflags="-X panel-api/internal/config.Version=1.2.3"
//
// If unset, falls back to "dev".
var Version = "dev"
