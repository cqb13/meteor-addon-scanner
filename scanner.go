package main

import (
	"dev/cqb13/meteor-addon-scanner/config"
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Failed to load env file: ", err)
		os.Exit(1)
	}

	// var key string = os.Getenv("KEY")

	config := config.ParseConfig()
}
