package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = "gatorconfig.json" // Use the file name in the project directory
const slashConfigFileName = "/.gatorconfig.json"

// Config represents the structure of the configuration file.
type Config struct {
	CurrentUserName string `json:"current_user_name"`
	DatabaseURL     string `json:"db_url"` // Add the DatabaseURL field
}

// readConfigFile reads and decodes the JSON file into a Config struct.
func readConfigFile(filePath string) (Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// If the file doesn't exist, return a default Config
			return Config{}, nil
		}
		return Config{}, err
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Read reads the JSON config file from the project directory and returns a Config struct.
func Read() (Config, error) {
	// Get the current working directory (i.e., the project directory)
	projectDir, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}

	// Build the full path to gatorconfig.json in the project directory
	filePath := filepath.Join(projectDir, configFileName)
	fmt.Println("Reading config from:", filePath) // Optional: Print the path for debugging
	return readConfigFile(filePath)
}

// SetUser sets the current user in the Config struct and writes the changes to the config file.
func (cfg *Config) SetUser(userName string) error {
	cfg.CurrentUserName = userName
	return cfg.write() // Call the write method on the Config struct
}

// write writes the Config struct to the JSON file in both project and home directories.
func (cfg *Config) write() error {
	fmt.Println(&cfg.CurrentUserName, &cfg.DatabaseURL)
	// Get the project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Build the path to the config file in the project directory
	projectFilePath := filepath.Join(projectDir, configFileName)
	fmt.Println("Writing to project directory:", projectFilePath)

	// Write to the config file in the project directory
	if err := writeToFile(cfg, projectFilePath); err != nil {
		return err
	}

	// Get the home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Build the path to the config file in the home directory
	homeFilePath := filepath.Join(homeDir, slashConfigFileName)
	fmt.Println("Writing to home directory:", homeFilePath)

	// Write to the config file in the home directory
	if err := writeToFile(cfg, homeFilePath); err != nil {
		return err
	}

	return nil
}

// writeToFile is a helper function to write the Config struct to a specified file path.
func writeToFile(cfg *Config, filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print the JSON.
	return encoder.Encode(cfg)
}
