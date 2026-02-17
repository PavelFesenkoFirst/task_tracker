package main

import (
	"log/slog"
	"os"
)

func main() {
	slog.Error("cmd/server is deprecated; use cmd/api instead")
	os.Exit(1)
}
