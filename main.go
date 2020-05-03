package main

import (
	inc "./include"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const Port = "55460"
const Version = "0.0.1"

func main() {

	inc.Db, inc.Err = gorm.Open("sqlite3", "pmi_tests.db")
	if inc.Err != nil {
		panic("ERROR failed to connect database")
	}
	defer inc.Db.Close()

	err := godotenv.Load("p.env")
	if err != nil {
		log.Fatal("ERROR loading .env file")
	}

	inc.Db.LogMode(false)

	inc.Db.AutoMigrate(
		&inc.Token{},
		&inc.Test{},
		&inc.TestError{},
	)

	if os.Getenv("MODE") == "DEBUG" {
		inc.DoNoticesInJsonTest("DebugRun")
	} else {
		go func() {

			var interval int
			interval_str := os.Getenv("SYNCRONIZATION_INTERVAL")
			if interval_str != "" {
				interval, err = strconv.Atoi(interval_str)
				if err != nil {
					interval = 30
				}
			} else {
				interval = 30
			}

			inc.DoEvery(time.Duration(interval)*time.Minute, inc.NoticesInJsonTest)

		}()
	}

	handleHTTP()
}

func handleHTTP() {

	r := mux.NewRouter()
	r.Use(inc.AuthMiddleware)
	r.Use(inc.HeaderMiddleware)

	r.HandleFunc("/test", inc.GetTestsStatistics)
	r.HandleFunc("/test/{id}", inc.GetTestStatistics)

	fmt.Printf("Starting Server to HANDLE pmi-test.maxtv.tech back end\nPort : " + Port + "\nAPI revision " + Version + "\n\n")
	if err := http.ListenAndServe(":"+Port, r); err != nil {
		log.Fatal(err, "ERROR")
	}
}
