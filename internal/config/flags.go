package config

import (
	"flag"
)

type Flags struct {
	Address         string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	SecretKey       string
}

func ParseFlags(args []string) (*Flags, error) {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	f := Flags{}

	fs.StringVar(&f.Address, "a", "localhost:8080", "server listen address")
	fs.IntVar(&f.StoreInterval, "i", 300, "interval to store data")
	fs.StringVar(&f.FileStoragePath, "f", "runtime_metrics.json", "file for runtime metrics")
	fs.BoolVar(&f.Restore, "r", false, "restore data from file")
	fs.StringVar(&f.DatabaseDSN, "d", "", "database DSN")
	fs.StringVar(&f.SecretKey, "k", "", "secret key")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &f, nil
}
