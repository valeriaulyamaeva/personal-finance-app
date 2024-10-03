package main

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"log"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	conn, err := database.ConnectDB()
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer conn.Close(context.Background())

	_, err = conn.Query(context.Background(), "SELECT 1")
	if err != nil {
		log.Fatal("Error executing test query:", err)
	}

	log.Println("Database connection established and test query successful")

}
