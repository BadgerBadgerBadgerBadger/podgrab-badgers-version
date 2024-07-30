package db

import (
	"fmt"
	"log"
	"os"
	"path"

	pkgErrors "github.com/pkg/errors"
	"gorm.io/driver/sqlite"

	"gorm.io/gorm"
)

// DB is
var DB *gorm.DB

// Init is used to Initialize Database
func Init() (*gorm.DB, error) {
	// github.com/mattn/go-sqlite3
	configPath := os.Getenv("CONFIG")
	dbPath := path.Join(configPath, "podgrab.db")
	log.Println(dbPath)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		fmt.Println("db err: ", err)
		return nil, err
	}

	localDB, _ := db.DB()
	localDB.SetMaxIdleConns(10)
	DB = db
	return DB, nil
}

// Migrate Database
func Migrate() error {
	err := DB.AutoMigrate(&Podcast{}, &PodcastItem{}, &Setting{}, &Migration{}, &JobLock{}, &Tag{})
	if err != nil {
		return pkgErrors.Wrap(err, "failed to migrate database")
	}
	RunMigrations()
	return nil
}
