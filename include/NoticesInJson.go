package include

import (
	"github.com/jinzhu/gorm"
	"log"
	"os"
	"strconv"
	"time"
)

type Test struct {
	gorm.Model
	Type        string
	Status      string
	Description string
	Duration    int
	ErrorCount  int
	RunType     string
}

type TestError struct {
	gorm.Model
	TestId      uint
	Type        string
	Message     string
	Description string
}

func NoticesInJsonTest(t time.Time) {
	DoNoticesInJsonTest("auto")
}

func DoNoticesInJsonTest(run_type string) {
	start := time.Now()

	var test Test
	test.RunType = run_type
	Db.Create(&test)

	U := os.Getenv("USER")
	P := os.Getenv("PASSWORD")

	Playlists := GetAllPlaylists(U, P)
	if Playlists == nil {

		test.Status = "ERROR"
		test.Description = "Unable to get List of Playlist"

	} else {

		for _, playlist := range Playlists {

			log.Println("###########################################################################")
			log.Println("Playlist Title : '"+playlist.Title+"' Announcements Count : ", playlist.Announcements)

			if playlist.Announcements == 0 {
				log.Println("Skipped, there is no Announcements")
				continue
			}
			DBNotices := GetAllNoticesByPlaylist(playlist.Id, U, P)
			if DBNotices == nil {
				var NoticeError TestError
				NoticeError.TestId = test.ID
				NoticeError.Type = "GetNoticesFromPMIError"
				NoticeError.Message = "Playlist :'" + playlist.Title + "' error getting Notices"
				Db.Create(&NoticeError)
				test.ErrorCount += 1
				continue
			}

			ServerNotices := GetServerPlaylistJson(playlist.Id)
			if ServerNotices == nil {
				var NoticeError TestError
				NoticeError.TestId = test.ID
				NoticeError.Type = "GetNoticesFromServerError"
				NoticeError.Message = "Playlist :'" + playlist.Title + "' (" + strconv.Itoa(playlist.Id) + ") error getting json From Server"
				Db.Create(&NoticeError)
				test.ErrorCount += 1
				continue
			}

			var NoticeFound bool
			for _, DBNotice := range DBNotices {
				if DBNotice.Status == "active" {
					NoticeFound = false
					log.Println("==================================================")
					log.Println(DBNotice.Title)
					log.Println("Start :", DBNotice.Schedule.ActivateFrom)
					log.Println("End :", DBNotice.Schedule.ActivateTo)
					for _, ServerNotice := range ServerNotices {
						if ServerNotice.PageId == DBNotice.Id {
							NoticeFound = true
							break
						}
					}

					if !NoticeFound {
						log.Println("Notice '" + DBNotice.Title + "' not found in JSON")

						linktonotice := os.Getenv("PMI_NOTICE_URL") + "#/notices/edit/" + strconv.Itoa(DBNotice.Id) + "/message"

						var NoticeError TestError
						NoticeError.TestId = test.ID
						NoticeError.Type = "NoticeJSONError"
						NoticeError.Message = "Playlist ID : " + strconv.Itoa(playlist.Id) + ", Notice ID : " + strconv.Itoa(DBNotice.Id) +
							" is not found in JSON, link to notice " + linktonotice

						Db.Create(&NoticeError)
						test.ErrorCount += 1
					}
				}
			}
		}
		test.Status = "Completed"
	}

	test.Duration = int(time.Since(start).Seconds())
	Db.Model(&Test{}).Update(&test)

}
