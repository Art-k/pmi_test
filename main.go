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

	//inc.Db, inc.Err = gorm.Open("sqlite3", "pmi_backup_29-06-2020.db")
	inc.Db, inc.Err = gorm.Open("sqlite3", "pmi.db")
	if inc.Err != nil {
		panic("ERROR failed to connect database")
	}
	defer inc.Db.Close()

	err := godotenv.Load("p.env")
	if err != nil {
		log.Fatal("ERROR loading .env file")
	}

	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("ERROR opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

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
		&inc.PodReplicas{},
		&inc.PodStat{},
		&inc.PodCpuMax{},
		&inc.PodRamMax{},
		&inc.PodStatNumber{},
		&inc.PodCpuMaxByHour{},
		&inc.CheckedNotices{},
		&inc.AbsentInJsonNotices{},
		&inc.TNoticesDiff{},
		&inc.PMINotices{},
		&inc.TPlayListsDiff{},
		&inc.PmiPlayList{},
		&inc.ActivationHistory{},
		&inc.TPlayListStat{},
		&inc.GetPlayListStats{},
		&inc.NoticeInPlaylist{},
	)

	go func() {
		if os.Getenv("MODE") == "DEBUG" {
			inc.DoNoticesInJsonTest("DebugRun")
			//inc.CompareStatusesCopiedNotices("debug")
		} else {
			//log.SetOutput(ioutil.Discard)

			go func() {
				inc.WL("== Set timer for Playlist Last Activity (PLA)==")
				var interval int
				interval_str := os.Getenv("SEND_STAT_INTERVAL")
				if interval_str != "" {

					interval, err = strconv.Atoi(interval_str)
					if err != nil {
						interval = 30
					}
				} else {
					interval = 30
				}
				inc.WL("PLA | " + strconv.Itoa(interval) + " minutes interval")
				inc.DoEvery(time.Duration(interval)*time.Minute, inc.GetPlayListStatByTimer)
			}()

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

			go func() {
				var interval int
				interval_str := os.Getenv("CLASSIFIEDS_INTERVAL")
				if interval_str != "" {
					interval, err = strconv.Atoi(interval_str)
					if err != nil {
						interval = 240
					}
				} else {
					interval = 240
				}

				inc.DoEvery(time.Duration(interval)*time.Minute, inc.CheckClassifieds)

			}()

			go func() {
				var interval int
				interval_str := os.Getenv("LAST_ACTIVITY_INTERVAL")
				if interval_str != "" {
					interval, err = strconv.Atoi(interval_str)
					if err != nil {
						interval = 240
					}
				} else {
					interval = 240
				}
				inc.DoEvery(time.Duration(interval)*time.Minute, inc.GetLastActivity)
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
	r.HandleFunc("/fixes/{test_id}", inc.GetFixesStatistics)
	r.HandleFunc("/diff/{notice_id}", inc.GetDiffs)
	r.HandleFunc("/test-result/{id}", inc.GetTestResult)
	r.HandleFunc("/ignore-pl", inc.IgnoredPlayLists)

	r.HandleFunc("/copy-notes", inc.CopyNotes)
	r.HandleFunc("/copy-notes/history/{id}", inc.FuncHistory)
	r.HandleFunc("/copy-notes/{id}", inc.CopyNotesTask)
	r.HandleFunc("/playlists-array", inc.GetAllPlaylistsAsArrayOfId)
	r.HandleFunc("/playlists", inc.GetAllNoticesGroupBy)
	r.HandleFunc("/used-copy", inc.GetUsedCopy)
	r.HandleFunc("/active-copy", inc.GetActiveCopy)
	r.HandleFunc("/compare-tasks", inc.GetComparesTasks)
	r.HandleFunc("/fix-playlists", inc.FixPlaylists)
	r.HandleFunc("/update-playlists", inc.UpdatePlaylists)

	r.HandleFunc("/mcc-docker-monitor", inc.APIMccDockerMonitor)
	r.HandleFunc("/mcc-docker-monitor-replicas", inc.APIMccDockerMonitorReplicas)
	r.HandleFunc("/mcc-docker-monitor-cpu-max", inc.APIMccDockerMonitorCpuMax)
	r.HandleFunc("/mcc-docker-monitor-ram-max", inc.APIMccDockerMonitorRamMax)
	r.HandleFunc("/mcc-docker-monitor-replica-max", inc.APIMccDockerMonitorReplicaMax)

	r.HandleFunc("/history/do-compare", inc.HistoryDoCompare)

	fmt.Printf("Starting Server to HANDLE pmi-test.maxtv.tech back end\nPort : " + Port + "\nAPI revision " + Version + "\n\n")
	if err := http.ListenAndServe(":"+Port, r); err != nil {
		log.Fatal(err, "ERROR")
	}
}
