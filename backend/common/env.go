package common

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	AppPort     int
	DataDir     string
	PluginsDir  string
	DownloadDir string
)

func InitEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Can't load .env file:", err)
	}
	AppPort = getEnvAsInt("APP_PORT", 8000)
	log.Printf("Using application port: %d\n", AppPort)

	DataDir = getEnvOrDefault("DATA_DIR", defaultDataDir())
	log.Printf("Using data directory: %s", DataDir)

	defaultPluginsDir, err := defaultPluginsDir()
	if err != nil {
		log.Printf("Can't determine user plugins directory: %v, using fallback", err)
		defaultPluginsDir = filepath.Join(DataDir, "plugins")
	}
	PluginsDir = getEnvOrDefault("PLUGINS_DIR", defaultPluginsDir)
	log.Printf("Using plugins directory: %s", PluginsDir)

	defaultDownloadsDir, err := defaultDownloadDir()
	if err != nil {
		log.Printf("Can't determine user download directory: %v, using fallback", err)
		defaultDownloadsDir = filepath.Join(DataDir, "downloads")
	}
	DownloadDir = getEnvOrDefault("DOWNLOAD_DIR", defaultDownloadsDir)
	log.Printf("Using download directory: %s", DownloadDir)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	log.Printf("Environment variable %s not set, using default value %s", key, defaultValue)
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnvOrDefault(key, strconv.Itoa(defaultValue))
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Fatalf("Invalid value for %s: %s. Must be an integer. Exiting...", key, valueStr)
	}
	return value
}

func defaultDataDir() string {
	switch runtime.GOOS {
	case "windows":
		if dir := os.Getenv("ProgramData"); dir != "" {
			return filepath.Join(dir, "MangaPull")
		}
		return `C:\ProgramData\MangaPull`

	case "darwin":
		return "/Library/Application Support/MangaPull"

	default:
		return "/var/lib/mangapull"
	}
}

func defaultPluginsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "MangaPull", "plugins"), nil
}

func defaultDownloadDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "MangaPull", "downloads"), nil
}
