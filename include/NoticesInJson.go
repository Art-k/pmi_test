package include

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type NoticeInJsonTestState struct {
	State   bool
	LastRun time.Time
}

var NoticeInJsonTestIsRunning NoticeInJsonTestState

type AbsentInJsonNotices struct {
	gorm.Model
	TestId          uint
	NoticeId        int
	Fixed           bool
	NoticeBeforeFix string
	NoticeAfterFix  string
}

func NoticesInJsonTest(t time.Time) {
	if !NoticeInJsonTestIsRunning.State {
		DoNoticesInJsonTest("auto")
	}
}

func DoNoticesInJsonTest(run_type string) {

	NoticeInJsonTestIsRunning.State = true
	NoticeInJsonTestIsRunning.LastRun = time.Now()

	start := time.Now()

	var test Test
	test.RunType = run_type
	Db.Create(&test)

	U := os.Getenv("PMI_USER")
	P := os.Getenv("PASSWORD")

	WL("(NIJ) | Get All Playlists")
	Playlists := GetAllPlaylists(U, P)
	if Playlists == nil {

		test.Status = "ERROR"
		test.Description = "Unable to get List of Playlist"
		WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Unable to get PlayList from server")

	} else {
		WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Get Playlists which are ignored")
		ignoredPlaylists := GetIgnoredPlaylists()

		for _, playlist := range Playlists {

			if os.Getenv("DEBUG_NOTICE_UPDATE_PLAYLIST") != "" {
				// DEBUG NOTICE, playlist is 175
				WL("(NIJ) | We are in DEBUG MODE")
				if strconv.Itoa(playlist.Id) != os.Getenv("DEBUG_NOTICE_UPDATE_PLAYLIST") {
					continue
				}
			}
			// "NIJ" Not In JSON
			WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Playlist Title : '" + playlist.Title + "' Announcements Count : " + strconv.Itoa(playlist.Announcements) + " ID :" + strconv.Itoa(playlist.Id))
			//log.Println("###########################################################################")
			//log.Println("Playlist Title : '"+playlist.Title+"' Announcements Count : ", playlist.Announcements, " ID :", strconv.Itoa(playlist.Id))

			if playlist.Announcements == 0 {
				WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Skipped, there is no announcements")
				//log.Println("Skipped, there is no Announcements")
				continue
			}

			if IfExists(ignoredPlaylists, playlist.Id) {
				WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Skipped, playlist marked as IGNORE")
				test.PlayListsIgnored++
				continue
			}

			WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Get All notices from playlist")
			DBNotices := GetAllNoticesByPlaylist(playlist.Id, U, P)
			if DBNotices == nil {
				var NoticeError TestError
				NoticeError.TestId = test.ID
				NoticeError.Type = "GetNoticesFromPMIError"
				NoticeError.Message = "Playlist :'" + playlist.Title + "' error getting Notices"
				Db.Create(&NoticeError)
				test.ErrorCount += 1
				WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Error getting Notices")
				continue
			} else {
				//log.Println("Found in DB " + strconv.Itoa(len(DBNotices)) + " notices")
				WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Found in DB " + strconv.Itoa(len(DBNotices)) + " notices")
			}

			WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Check if at least one notice is active")
			var activeIsHere bool
			for _, notice := range DBNotices {
				if notice.Status == "active" {
					activeIsHere = true
					break
				}
			}

			if activeIsHere {
				WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | We have active notices, lets check JSON")
				ServerNotices := GetServerPlaylistJson(playlist.Id)
				if ServerNotices == nil {
					var NoticeError TestError

					NoticeError.TestId = test.ID
					NoticeError.Type = "GetNoticesFromServerError"
					NoticeError.Message = "Playlist :'" + playlist.Title + "' (" + strconv.Itoa(playlist.Id) + ") error getting json From Server"
					Db.Create(&NoticeError)
					test.ErrorCount += 1
					WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | can't get json from server")
					continue
				} else {
					log.Println("Found on server " + strconv.Itoa(len(ServerNotices)) + " notices")
					WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Found on server " + strconv.Itoa(len(ServerNotices)) + " notices")
				}

				WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Compare DB and JSON")

				var NoticeFound bool
				var TimeIsCorrect bool
				for _, DBNotice := range DBNotices {
					if DBNotice.Status == "active" {
						NoticeFound = false
						TimeIsCorrect = true
						log.Println("==================================================")
						log.Println(DBNotice.Title)
						log.Println("Start :", DBNotice.Schedule.ActivateFrom)
						log.Println("End :", DBNotice.Schedule.ActivateTo)
						for _, ServerNotice := range ServerNotices {
							if ServerNotice.PageId == DBNotice.Id {
								NoticeFound = true

								var err error
								var serverNoticeStart time.Time
								var pmiNoticeStart time.Time
								var serverNoticeEnd time.Time
								var pmiNoticeEnd time.Time

								if DBNotice.Schedule.ActivateFrom != "0000-00-00 00:00:00" {

									serverNoticeStart, err = time.Parse("2006-01-02 15:04:05", ServerNotice.ActivateStart)
									if err != nil {
										fmt.Printf("serverNoticeStart Got an error %s ", err)
									}

									pmiNoticeStart, err = time.Parse("2006-01-02 15:04:05", DBNotice.Schedule.ActivateFrom)
									if err != nil {
										fmt.Printf("pmiNoticeStart Got an error %s ", err)
									}

									if ServerNotice.ActivateStart != DBNotice.Schedule.ActivateFrom {
										correctedTime := serverNoticeStart.Add(time.Duration(DBNotice.Schedule.LocalTimeOffset) * time.Minute)
										if !pmiNoticeStart.Equal(correctedTime) {
											TimeIsCorrect = false
										}
									}

									if DBNotice.Schedule.ActivateTo != "0000-00-00 00:00:00" && ServerNotice.ActivateEnd != "2030-01-01 00:00:00" {

										serverNoticeEnd, err = time.Parse("2006-01-02 15:04:05", ServerNotice.ActivateStart)
										if err != nil {
											fmt.Printf("Got an error %s ", err)
										}

										pmiNoticeEnd, err = time.Parse("2006-01-02 15:04:05", DBNotice.Schedule.ActivateFrom)
										if err != nil {
											fmt.Printf("Got an error %s ", err)
										}

										if ServerNotice.ActivateEnd != DBNotice.Schedule.ActivateTo {
											correctedTime := serverNoticeEnd.Add(time.Duration(DBNotice.Schedule.LocalTimeOffset) * time.Minute)
											if !pmiNoticeEnd.Equal(correctedTime) {
												TimeIsCorrect = false
											}
										}
									}
								}

								break
							}
						}

						if !NoticeFound || !TimeIsCorrect {
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

							if !TimeIsCorrect {
								//var NoticeError TestError
								//NoticeError.TestId = test.ID
								//NoticeError.Type = "NoticeJSONTimeError"

								NoticeError.Message = "Playlist :'" + playlist.Title + "' (" + strconv.Itoa(playlist.Id) + "), Notice ID : " + strconv.Itoa(DBNotice.Id) +
									" time is not correct in JSON, link to notice"

								PostTelegrammMessage(NoticeError.Message)

								//Db.Create(&NoticeError)
								test.ErrorCount += 1
							}

							if NoticeError.ID != 0 {
								if !NoticeFound {

									NoticeError.TestId = test.ID
									NoticeError.Type = "NoticeJSONError"

									NoticeError.Message = "Playlist :'" + playlist.Title + "' (" + strconv.Itoa(playlist.Id) + "), Notice ID : " + strconv.Itoa(DBNotice.Id) +
										" is not found in JSON, link to notice <a href=\"" + strconv.Itoa(DBNotice.Id) + "\">" + linktonotice + "</a>"

									Db.Create(&NoticeError)
									test.ErrorCount += 1
								}
							}

						}
					}
				}
				test.PlayListsTested++
			}
		}
		WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Test completed")
		test.Status = "Completed"
	}

	test.Duration = int(time.Since(start).Seconds())
	test.Hash = GetHash()

	if test.ErrorCount != 0 {

		//PostTelegrammMessage("TEST " + strconv.Itoa(int(test.ID)) + ", Notices in JSON found " + strconv.Itoa(test.ErrorCount) + " errors, listed below. Please find it here [link](https://pmi-test.maxtv.tech/test-result/" + test.Hash + ")")
		go FixAbsentNotices(test.ID)

	}
	WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Update test task in DB")
	Db.Model(&Test{}).Save(&test)
	NoticeInJsonTestIsRunning.State = false
	WL("NIJ (" + strconv.Itoa(int(test.ID)) + ") | Flag NoticeInJsonTestIsRunning set to " + strconv.FormatBool(NoticeInJsonTestIsRunning.State))
}

func FixAbsentNotices(testId uint) {

	// TODO добавить обработку того что может быть 2 одинаковых ID в одном тесте

	U := os.Getenv("PMI_USER")
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
			if diffLength != 2 {
				PostTelegrammMessage("!!! ERROR Test ID:" + strconv.Itoa(int(testId)) + " Notice ID:" + strconv.Itoa(absentNotice.NoticeId) + " is updated but not the same")
			} else {
				if diff[0].FieldName != "EditedAt" {
					PostTelegrammMessage("!!! ERROR Test ID:" + strconv.Itoa(int(testId)) + " Notice ID:" + strconv.Itoa(absentNotice.NoticeId) + " is updated but not the same")
				}
			}
			//PostTelegrammMessage("Test ID:" + strconv.Itoa(int(testId)) + " Notice ID:" + strconv.Itoa(absentNotice.NoticeId) + " is updated")
			Db.Model(&AbsentInJsonNotices{}).Where("id = ?", absentNotice.ID).Update("fixed", updateResult)
			Db.Model(&AbsentInJsonNotices{}).Where("id = ?", absentNotice.ID).Update("notice_after_fix", string(noticeStr))
		}
		time.Sleep(20 * time.Second)
	}
}

func UpdatePlaylists(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":

		U := os.Getenv("PMI_USER")
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
