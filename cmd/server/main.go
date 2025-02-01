package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Mort4lis/memdb/internal/db"
)

func main() {
	var confPath string

	flag.StringVar(&confPath, "c", "config.yaml", "The configuration file path")
	flag.Parse()

	if err := db.Run(confPath); err != nil {
		fmt.Fprintf(os.Stderr, "An error occurs while running the database: %v", err)
		os.Exit(1)
	}
}
