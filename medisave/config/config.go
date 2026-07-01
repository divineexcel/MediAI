package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Paystack PaystackConfig
	Maps     MapsConfig
	Firebase FirebaseConfig
	AI       AIConfig
	SMS      SMSConfig
	USSD     USSDConfig
	Rate     RateLimitConfig
}

type AppConfig struct {
	Env  string
	Port string
	URL  string
}

type DatabaseConfig struct {
	Driver string
	Path   string
	URL    string
}

type JWTConfig struct {
	AccessSecret        string
	RefreshSecret       string
	AccessExpiryHours   int
	RefreshExpiryDays   int
}

type PaystackConfig struct {
	SecretKey  string
	PublicKey  string
	BaseURL    string
}

type MapsConfig struct {
	APIKey       string
	PlacesAPIKey string
}

type FirebaseConfig struct {
	CredentialsPath string
}

type AIConfig struct {
	BaseURL        string
	APIKey         string
	TimeoutSeconds int
}

type SMSConfig struct {
	GatewayURL string
	APIKey     string
	Username   string
	SenderID   string
}

type USSDConfig struct {
	ServiceCode string
}

type RateLimitConfig struct {
	RequestsPerMinute int
}

var App *Config

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	App = &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "development"),
			Port: getEnv("APP_PORT", "8080"),
			URL:  getEnv("APP_URL", "http://localhost:8080"),
		},
		Database: DatabaseConfig{
			Driver: getEnv("DB_DRIVER", "sqlite"),
			Path:   getEnv("DB_PATH", "./data/medisave.db"),
			URL:    getEnv("DB_URL", ""),
		},
		JWT: JWTConfig{
			AccessSecret:      getEnv("JWT_ACCESS_SECRET", ""),
			RefreshSecret:     getEnv("JWT_REFRESH_SECRET", ""),
			AccessExpiryHours: getEnvInt("JWT_ACCESS_EXPIRY_HOURS", 24),
			RefreshExpiryDays: getEnvInt("JWT_REFRESH_EXPIRY_DAYS", 7),
		},
		Paystack: PaystackConfig{
			SecretKey: getEnv("PAYSTACK_SECRET_KEY", ""),
			PublicKey: getEnv("PAYSTACK_PUBLIC_KEY", ""),
			BaseURL:   getEnv("PAYSTACK_BASE_URL", "https://api.paystack.co"),
		},
		Maps: MapsConfig{
			APIKey:       getEnv("GOOGLE_MAPS_API_KEY", ""),
			PlacesAPIKey: getEnv("GOOGLE_PLACES_API_KEY", ""),
		},
		Firebase: FirebaseConfig{
			CredentialsPath: getEnv("FIREBASE_CREDENTIALS_PATH", "./config/firebase-service-account.json"),
		},
		AI: AIConfig{
			BaseURL:        getEnv("AI_API_BASE_URL", ""),
			APIKey:         getEnv("AI_API_KEY", ""),
			TimeoutSeconds: getEnvInt("AI_API_TIMEOUT_SECONDS", 30),
		},
		SMS: SMSConfig{
			GatewayURL: getEnv("SMS_GATEWAY_URL", ""),
			APIKey:     getEnv("SMS_API_KEY", ""),
			Username:   getEnv("SMS_USERNAME", "sandbox"),
			SenderID:   getEnv("SMS_SENDER_ID", "MediSave"),
		},
		USSD: USSDConfig{
			ServiceCode: getEnv("USSD_SERVICE_CODE", "*384*123#"),
		},
		Rate: RateLimitConfig{
			RequestsPerMinute: getEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 100),
		},
	}

	return App
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
