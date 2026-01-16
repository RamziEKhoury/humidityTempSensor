package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/RamziEKhoury/iotServer/db"
	"github.com/joho/godotenv"
)

var conn *sql.DB

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	port, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		log.Fatalf("Invalid DB_PORT: %v", err)
	}

	cfg := &db.DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     port,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
	}

	conn, err = db.OpenDB(cfg)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}
	defer conn.Close()

	if err := InitTemplates(); err != nil {
		log.Fatalf("failed to load templates: %v", err)
	}

	// Dashboard routes
	http.HandleFunc("/", HandleDashboard)
	http.HandleFunc("/devices", HandleDevices)
	http.HandleFunc("/device/new", HandleAddDeviceForm)
	http.HandleFunc("/device", HandleCreateDevice)
	http.HandleFunc("/device/", HandleDevice)

	// API routes
	http.HandleFunc("/weatherListener", PostWeatherListener)

	log.Println("Server starting on :8000")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
