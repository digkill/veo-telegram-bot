// internal/db/mysql.go
package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Connect() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_NAME"),
	)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("❌ Не удалось подключиться к MySQL: %v", err)
	}

	if err := DB.Ping(); err != nil {
		log.Fatalf("❌ MySQL недоступен: %v", err)
	}

	log.Println("✅ MySQL подключен успешно")
}
