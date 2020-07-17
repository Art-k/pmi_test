package include

import (
	"encoding/json"
	"github.com/jinzhu/gorm"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var NoticeInJsonTestIsRunning bool

type AbsentInJsonNotices struct {
	gorm.Model
	TestId          uint
	NoticeId        int
	Fixed           bool `gorm:"default:'false'"`
	NoticeBeforeFix string
	NoticeAfterFix  string
}

func NoticesInJsonTest(t time.Time) {
	if !NoticeInJsonTestIsRunning {
		DoNoticesInJsonTest("auto")
	}
}

func DoNoticesInJsonTest(run_type string) {

	NoticeInJsonTestIsRunning = true

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

		ignoredPlaylists := GetIgnoredPlaylists()

		for _, playlist := range Playlists {

			if os.Getenv("DEBUG_NOTICE_UPDATE_PLAYLIST") != "" {
				// DEBUG NOTICE, playlist is 175
				if strconv.Itoa(playlist.Id) != os.Getenv("DEBUG_NOTICE_UPDATE_PLAYLIST") {
					continue
				}
			}

			log.Println("###########################################################################")
			log.Println("Playlist Title : '"+playlist.Title+"' Announcements Count : ", playlist.Announcements, " ID :", strconv.Itoa(playlist.Id))

			if playlist.Announcements == 0 {
				log.Println("Skipped, there is no Announcements")
				continue
			}

			if IfExists(ignoredPlaylists, playlist.Id) {
				test.PlayListsIgnored++
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
			} else {
				log.Println("Found in DB " + strconv.Itoa(len(DBNotices)) + " notices")
			}

			var activeIsHere bool
			for _, notice := range DBNotices {
				if notice.Status == "active" {
					activeIsHere = true
					break
				}
			}

			if activeIsHere {
				ServerNotices := GetServerPlaylistJson(playlist.Id)
				if ServerNotices == nil {
					var NoticeError TestError

					NoticeError.TestId = test.ID
					NoticeError.Type = "GetNoticesFromServerError"
					NoticeError.Message = "Playlist :'" + playlist.Title + "' (" + strconv.Itoa(playlist.Id) + ") error getting json From Server"
					Db.Create(&NoticeError)
					test.ErrorCount += 1
					continue
				} else {
					log.Println("Found on server " + strconv.Itoa(len(ServerNotices)) + " notices")
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

							var absentNotice AbsentInJsonNotices
							Db.Where("test_id = ?", test.ID).Where("notice_id = ?", DBNotice.Id).Find(&absentNotice)
							if absentNotice.ID == 0 {
								absentNotice.TestId = test.ID
								absentNotice.NoticeId = DBNotice.Id
								Db.Create(&absentNotice)
							}

							var NoticeError TestError
							NoticeError.TestId = test.ID
							NoticeError.Type = "NoticeJSONError"

							NoticeError.Message = "Playlist :'" + playlist.Title + "' (" + strconv.Itoa(playlist.Id) + "), Notice ID : " + strconv.Itoa(DBNotice.Id) +
								" is not found in JSON, link to notice <a href=\"" + strconv.Itoa(DBNotice.Id) + "\">" + linktonotice + "</a>"

							Db.Create(&NoticeError)
							test.ErrorCount += 1
						}
					}
				}
				test.PlayListsTested++
			}
		}
		test.Status = "Completed"
	}

	test.Duration = int(time.Since(start).Seconds())
	test.Hash = GetHash()

	if test.ErrorCount != 0 {

		PostTelegrammMessage("TEST " + strconv.Itoa(int(test.ID)) + ", Notices in JSON found " + strconv.Itoa(test.ErrorCount) + " errors, listed below. Please find it here [link](https://pmi-test.maxtv.tech/test-result/" + test.Hash + ")")
		go FixAbsentNotices(test.ID)

	}

	Db.Model(&Test{}).Update(&test)

	NoticeInJsonTestIsRunning = false
}

func FixAbsentNotices(testId uint) {

	U := os.Getenv("USER")
	P := os.Getenv("PASSWORD")

	var noticesToFix []AbsentInJsonNotices
	Db.Where("test_id = ?", testId).Find(&noticesToFix)
	for _, absentNotice := range noticesToFix {
		var reference TypeNotice
		reference = GetNoticeById(absentNotice.NoticeId, U, P)
		referenceStr, _ := json.Marshal(&reference)

		Db.Model(&AbsentInJsonNotices{}).Where("id = ?", absentNotice.ID).Update("notice_before_fix", string(referenceStr))
		updateResult := UpdateNoticeById(absentNotice.NoticeId, reference, U, P)

		if updateResult {
			notice := GetNoticeById(absentNotice.NoticeId, U, P)
			noticeStr, _ := json.Marshal(&reference)
			diff, diffLength := Compare2Notices(reference, notice)
			if diffLength != 1 {
				PostTelegrammMessage("!!! ERROR Test ID:" + strconv.Itoa(int(testId)) + " Notice ID:" + strconv.Itoa(absentNotice.NoticeId) + " is updated but not the same")
			} else {
				if diff[0].FieldName != "EditedAt" {
					PostTelegrammMessage("!!! ERROR Test ID:" + strconv.Itoa(int(testId)) + " Notice ID:" + strconv.Itoa(absentNotice.NoticeId) + " is updated but not the same")
				}
			}
			PostTelegrammMessage("Test ID:" + strconv.Itoa(int(testId)) + " Notice ID:" + strconv.Itoa(absentNotice.NoticeId) + " is updated")
			Db.Model(&AbsentInJsonNotices{}).Where("id = ?", absentNotice.ID).Update("fixed", updateResult)
			Db.Model(&AbsentInJsonNotices{}).Where("id = ?", absentNotice.ID).Update("notice_after_fix", string(noticeStr))
		}

		time.Sleep(20 * time.Second)
	}

}

func UpdatePlaylists(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":

		U := os.Getenv("USER")
		P := os.Getenv("PASSWORD")
		taskHash := GetHash()
		type DoUpdate struct {
			PlayLists    []int
			SleepSeconds int
		}

		var incomingData DoUpdate
		err := json.NewDecoder(r.Body).Decode(&incomingData)
		if err != nil {
			ResponseBadRequest(w, err, "")
			return
		}
		PostTelegrammMessage(taskHash + "We need to update " + strconv.Itoa(len(incomingData.PlayLists)) + " plylists")

		go func() {
			for ind, playlistId := range incomingData.PlayLists {
				ActiveNotices := false
				notices := GetAllNoticesByPlaylist(playlistId, U, P)
				for _, notice := range notices {
					if notice.Status == "Active" {
						ActiveNotices = true
						n := GetNoticeById(notice.Id, U, P)
						status := UpdateNoticeById(n.Id, n, U, P)
						aUpdate := GetNoticeById(notice.Id, U, P)
						Compare2Notices(n, aUpdate)
						if status {
							PostTelegrammMessage(taskHash + "Playlist ID: " + strconv.Itoa(playlistId) + " is updated (" +
								strconv.Itoa(len(incomingData.PlayLists)-ind-1) + " left) waiting " +
								strconv.Itoa(incomingData.SleepSeconds) + " seconds")
						} else {
							PostTelegrammMessage(taskHash + "!!! UPDATE ERROR Playlist ID: " + strconv.Itoa(playlistId) + " (" +
								strconv.Itoa(len(incomingData.PlayLists)-ind-1) + " left) waiting " +
								strconv.Itoa(incomingData.SleepSeconds) + " seconds")
						}
					}
				}
				if !ActiveNotices {
					PostTelegrammMessage(taskHash + "Playlist ID: " + strconv.Itoa(playlistId) + " (" +
						strconv.Itoa(len(incomingData.PlayLists)-ind-1) + " left) waiting " +
						strconv.Itoa(incomingData.SleepSeconds) + " seconds")
				}
				log.Println("waiting")
				time.Sleep(time.Duration(incomingData.SleepSeconds) * time.Second)
			}
		}()

	default:
		ResponseUnknown(w, "Method is unknown")
	}
}
