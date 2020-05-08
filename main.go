package main

import (
	inc "./include"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const Port = "55460"
const Version = "0.0.1"

func main() {

	inc.Db, inc.Err = gorm.Open("sqlite3", "pmi.db")
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
		&inc.CopyNoticesToPlaylistsTask{},
		&inc.DestinationPlaylists{},
		&inc.IgnoredPlaylist{},
		&inc.CopiedNoticesHistory{},
		&inc.ComparesTaskType{},
	)

	go func() {
		if os.Getenv("MODE") == "DEBUG" {
			inc.DoNoticesInJsonTest("DebugRun")
			inc.CompareStatusesCopiedNotices("debug")
		} else {
			log.SetOutput(ioutil.Discard)
			go func() {

				var interval int
				interval_str := os.Getenv("TEST_INTERVAL")
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

			go func() {
				var interval int
				interval_str := os.Getenv("HISTORY_INTERVAL")
				if interval_str != "" {
					interval, err = strconv.Atoi(interval_str)
					if err != nil {
						interval = 240
					}
				} else {
					interval = 240
				}

				inc.DoEvery(time.Duration(interval)*time.Minute, inc.MakeHistory)

			}()
		}
	}()

	handleHTTP()
}

func handleHTTP() {

	r := mux.NewRouter()
	r.Use(inc.AuthMiddleware)
	r.Use(inc.HeaderMiddleware)

	r.HandleFunc("/test", inc.GetTestsStatistics)
	r.HandleFunc("/test/{id}", inc.GetTestStatistics)
	r.HandleFunc("/test-result/{id}", inc.GetTestResult)
	r.HandleFunc("/ignore-pl", inc.IgnoredPlayLists)

	r.HandleFunc("/copy-notes", inc.CopyNotes)
	r.HandleFunc("/copy-notes/history", inc.FuncHistory)
	r.HandleFunc("/copy-notes/{id}", inc.CopyNotesTask)
	r.HandleFunc("/playlists-array", inc.GetAllPlaylistsAsArrayOfId)
	r.HandleFunc("/used-copy", inc.GetUsedCopy)
	r.HandleFunc("/active-copy", inc.GetActiveCopy)
	r.HandleFunc("/compare-tasks", inc.GetComparesTasks)

	fmt.Printf("Starting Server to HANDLE pmi-test.maxtv.tech back end\nPort : " + Port + "\nAPI revision " + Version + "\n\n")
	if err := http.ListenAndServe(":"+Port, r); err != nil {
		log.Fatal(err, "ERROR")
	}
}
