package main

import (
	"fmt"
	"log"

	"github.com/LandcLi/landc-go/mvc/pkg/config"
)

func RunConfigExample() {
	fmt.Println("=== Config Example ===\n")

	fmt.Println("1. Using default config:")
	err := config.InitGlobalConfigWithDefault()
	if err != nil {
		log.Fatalf("Failed to init default config: %v", err)
	}
	cfg := config.GetConfig()
	fmt.Printf("Server: %s\n", cfg.GetServerAddr())
	fmt.Printf("Database: %s\n", cfg.Database.Driver)
	fmt.Printf("Log Level: %s\n", cfg.Log.Level)

	fmt.Println("\n2. Using custom config:")
	customConfig := &config.Config{
		Server: config.ServerConfig{
			Addr: "127.0.0.1",
			Port: 9000,
		},
		Database: config.DatabaseConfig{
			Driver: "postgres",
			DSN:    "postgres://user:pass@localhost/db",
		},
	}
	err = config.InitGlobalConfigWithConfig(customConfig)
	if err != nil {
		log.Fatalf("Failed to init custom config: %v", err)
	}
	cfg = config.GetConfig()
	fmt.Printf("Server: %s\n", cfg.GetServerAddr())
	fmt.Printf("Database: %s\n", cfg.Database.Driver)

	fmt.Println("\n3. Validating config:")
	err = cfg.Validate()
	if err != nil {
		fmt.Printf("Config validation failed: %v\n", err)
	} else {
		fmt.Println("Config is valid")
	}

	fmt.Println("\n4. Exporting config:")
	jsonData, err := cfg.ToJSON()
	if err != nil {
		log.Fatalf("Failed to export JSON: %v", err)
	}
	fmt.Printf("JSON:\n%s\n", string(jsonData))

	yamlData, err := cfg.ToYAML()
	if err != nil {
		log.Fatalf("Failed to export YAML: %v", err)
	}
	fmt.Printf("YAML:\n%s\n", string(yamlData))
}
