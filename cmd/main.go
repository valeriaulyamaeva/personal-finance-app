package main

import (
	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-appersonal-/internal/database"
	"log"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	conn, err := database.ConnectDB()
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}
	defer conn.Close()

}
