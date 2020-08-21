package include

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type ActivationHistory struct {
	Model
	PlayListID int
	NoticeId   int
}

type Model struct {
	ID        string `gorm:"primary_key"`
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
	DeletedAt *time.Time
	// DeletedBy string
}

func (base *Model) BeforeCreate(scope *gorm.Scope) error {
	// uuID, err := uuid.NewRandom()
	// if err != nil {
	// 	return err
	// }
	return scope.SetColumn("id", GetHash())
}

type PmiPlayList struct {
	TypePlaylist
}

type PMINotices struct {
	TypeNotice
}

func GetPlayListStatByTimer(t time.Time) {
	WL("PLA | The current task status is : " + strconv.FormatBool(CompareTaskIsActive))
	if !CompareTaskIsActive {
		WL("PLA | get last active changes")
		SaveNoticeChanges("by timer")
	}
}

func SendPlaylistActivityToCMS(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		params := mux.Vars(r)
		var recs []TPlayListStat
		Db.Where("task_id = ?", params["task_id"]).Find(&recs)
		if len(recs) == 0 {
			ResponseNotFound(w)
			return
		}
		var stats []PlayListStat
		for _, rec := range recs {
			var stat PlayListStat
			stat = rec.PlayListStat
			stats = append(stats, stat)
		}
		PostPlayListStatToCMS(params["task_id"], stats)

	default:
		ResponseBadRequest(w, nil, "Method not found")
	}
}
func GetLastPlaylistActivity(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		params := mux.Vars(r)
		var recs []TPlayListStat
		Db.Where("task_id = ?", params["task_id"]).Find(&recs)

		response, err := json.Marshal(recs)
		if err != nil {
			ResponseNotFound(w)
		}
		ResponseOK(w, response)

	default:
		ResponseBadRequest(w, nil, "Method not found")
	}

}

func HistoryDoCompare(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case "GET":

		var allTasks []GetPlayListStats
		Db.Order("created_at desc").Find(&allTasks)
		response, _ := json.Marshal(allTasks)
		ResponseOK(w, response)

	case "POST":

		if !CompareTaskIsActive {
			task := SaveNoticeChanges("over http")
			response, _ := json.Marshal(&task)
			ResponseOK(w, response)
		} else {
			ResponseBadRequest(w, nil, "Compare Task is in Progress")
		}

	default:
		ResponseBadRequest(w, nil, "Method not found")
	}

}

func SaveNoticeChanges(runType string) GetPlayListStats {

	var task GetPlayListStats
	task.RunType = runType
	task.Status = "In Progress"
	Db.Create(&task)
	WL("PLA | Task is created, task ID is " + task.ID)

	CompareTaskIsActive = true
	WL("PLA | Flag set to " + strconv.FormatBool(CompareTaskIsActive))

	go func(tId string) {
		WL("PLA (" + tId + ") | Start routine")
		var lastActiveNotices map[int]string
		lastActiveNotices = make(map[int]string)

		start := time.Now()

		var messageToCMS []PlayListStat

		U := os.Getenv("USER")
		P := os.Getenv("PASSWORD")

		WL("PLA (" + tId + ") | Get Playlists")
		playLists := GetAllPlaylists(U, P)
		WL("PLA (" + tId + ") | Analyze Playlists Changes")
		AnalyzePlaylistsChanges(playLists)

		var taskTmp GetPlayListStats
		Db.Where("id = ?", tId).Find(&taskTmp)
		taskTmp.PlayListCount = len(playLists)
		Db.Save(&taskTmp)
		WL("PLA (" + tId + ") | PlayList count Saved to DB " + strconv.Itoa(taskTmp.PlayListCount))

		WL("PLA (" + tId + ") | Go thought playlists and check notices")
		for _, pl := range playLists {

			if os.Getenv("MODE") != "LIVE" {
				fmt.Println(pl.Title)
			}

			var plLength int
			var plLengthCount int
			var plMaxLength int
			var plMinLength int
			plMinLength = 9999999999

			if os.Getenv("DEBUG_NOTICE_UPDATE_PLAYLIST") != "" {
				plId, _ := strconv.Atoi(os.Getenv("DEBUG_NOTICE_UPDATE_PLAYLIST"))
				if pl.Id != plId {
					continue
				}
			}

			WL("PLA (" + tId + ") | Playlist is '" + pl.Title + "' (" + strconv.Itoa(pl.Id) + ")")

			var playListStat PlayListStat
			playListStat.PlayListId = pl.Id
			playListStat.PlayListName = pl.Title

			WL("PLA (" + tId + ") | Get All Notices")
			notices := GetAllNoticesByPlaylist(pl.Id, U, P)
			for _, notice := range notices {

				var noticeInPlaylist NoticeInPlaylist
				var noticesInPlaylist []NoticeInPlaylist

				// HERE we are trying to remove error, it is temporary code TODO need to remove it in the future
				Db.Where("p_lay_list_id = ?", pl.Id).
					Where("notice_id = ?", notice.Id).Find(&noticesInPlaylist)
				if len(noticesInPlaylist) > 1 {
					log.Println("We need to remove ", len(noticesInPlaylist)-1, " duplicates")
					noticesInPlaylist = noticesInPlaylist[:len(noticesInPlaylist)-1]
					Db.Unscoped().Where("p_lay_list_id = ?", pl.Id).
						Where("notice_id = ?", notice.Id).Delete(&noticesInPlaylist)
				}

				Db.Where("p_lay_list_id = ?", pl.Id).
					Where("notice_id = ?", notice.Id).Find(&noticeInPlaylist)
				if noticeInPlaylist.ID == "" {
					noticeInPlaylist.PLayListId = pl.Id
					noticeInPlaylist.NoticeId = notice.Id
					Db.Create(&noticeInPlaylist)
				}

				WL("PLA (" + tId + ") | check Advance Scheduling")
				if notice.Schedule.Hours != "" || notice.Schedule.Days != "" || notice.Schedule.Month != "" || notice.Schedule.WeekDays != "" {
					playListStat.NumberOfAdvancedScheduled++
				}

				WL("PLA (" + tId + ") | check No Expiration Date")
				if notice.Schedule.ActivateTo == "0000-00-00 00:00:00" {
					playListStat.NumberOfForeverNotices++
				}

				WL("PLA (" + tId + ") | check is PDF")
				if notice.Pdf {
					playListStat.NumberOfPDFNotices++
				}

				WL("PLA (" + tId + ") | check Status")
				switch notice.Status {
				case StatusActive:
					playListStat.NumberOfActiveNotices++
					playListStat.TotalDurationSeconds += notice.Schedule.Duration

					if (notice.Schedule.ActivateFrom != "" && notice.Schedule.ActivateFrom != "0000-00-00 00:00:00") &&
						notice.Schedule.ActivateTo != "" && notice.Schedule.ActivateTo != "0000-00-00 00:00:00" {

						plLengthCount++

						from, _ := time.Parse("2006-01-02 15:04:05", notice.Schedule.ActivateFrom)
						to, _ := time.Parse("2006-01-02 15:04:05", notice.Schedule.ActivateTo)

						length := int(to.Sub(from).Hours() / 24)
						plLength += length
						if plMinLength > length {
							plMinLength = length
						}
						if plMaxLength < length {
							plMaxLength = length
						}
					}

				case StatusExpired:
					playListStat.NumberOfExpiredNotices++
				case StatusArchived:
					playListStat.NumberOfArchivedNotices++
				case StatusFuture:
					playListStat.NumberOfFutureNotices++
				default:
					log.Fatal("Unknown Status : '" + notice.Status + "'")
				}

				var dbNotice PMINotices
				Db.Unscoped().Where("id = ?", notice.Id).Find(&dbNotice)

				if dbNotice.Id != 0 {

					if val, ok := lastActiveNotices[notice.Id]; ok {
						playListStat.LastActivity = val
					} else {

						diff, _ := Compare2Notices(dbNotice.TypeNotice, notice)
						for _, di := range diff {
							if di.FieldName == "Status" {
								if di.RefStrValue != StatusActive && di.NoticeStrValue == StatusActive {
									Db.Create(&ActivationHistory{
										PlayListID: pl.Id,
										NoticeId:   notice.Id,
									})
									playListStat.LastActivity = time.Now().Format("2006-01-02 15:04:05")
									lastActiveNotices[notice.Id] = playListStat.LastActivity
								}
							}
						}
						if len(diff) != 0 {
							dbNotice.TypeNotice = notice
							Db.Save(&dbNotice)
						}
					}

				} else {
					dbNotice.TypeNotice = notice
					Db.Create(&dbNotice)
					if dbNotice.Status == "Active" {
						Db.Create(&ActivationHistory{
							PlayListID: pl.Id,
							NoticeId:   notice.Id,
						})
					}
					playListStat.LastActivity = time.Now().Format("2006-01-02 15:04:05")
					lastActiveNotices[notice.Id] = playListStat.LastActivity
				}
			}

			if plLengthCount != 0 {
				playListStat.AvgActiveDays = int(plLength / plLengthCount)
			}
			playListStat.MinActiveDays = plMinLength
			playListStat.MaxActiveDays = plMaxLength

			var dbPlayListStat TPlayListStat
			dbPlayListStat.TaskID = tId
			dbPlayListStat.PlayListStat = playListStat
			Db.Create(&dbPlayListStat)

			taskTmp.PlayListProcessed++
			Db.Save(&taskTmp)

			var noticesInPlayList []NoticeInPlaylist
			Db.Where("play_list_id = ?", pl.Id).Find(&noticesInPlayList)

			messageToCMS = append(messageToCMS, playListStat)

		}

		elapsed := time.Since(start).Seconds()
		var t GetPlayListStats
		Db.Where("id = ?", tId).Find(&t)
		t.DurationSec = int(elapsed)
		t.Status = "Completed"
		Db.Save(&t)

		CompareTaskIsActive = false

		PostPlayListStatToCMS(task.ID, messageToCMS)

	}(task.ID)

	return task
}

func AnalyzePlaylistsChanges(playLists []TypePlaylist) {

	var dbPlayLists []PmiPlayList
	Db.Find(&dbPlayLists)

	for _, pl := range playLists {

		if os.Getenv("DEBUG_NOTICE_UPDATE_PLAYLIST") != "" {
			plId, _ := strconv.Atoi(os.Getenv("DEBUG_NOTICE_UPDATE_PLAYLIST"))
			if pl.Id != plId {
				continue
			}
		}

		playListFound := false
		for _, dbPl := range dbPlayLists {
			if pl.Id == dbPl.Id {
				playListFound = true
				var dbPlayList TypePlaylist
				dbPlayList.Id = dbPl.Id
				dbPlayList.Title = dbPl.Title
				dbPlayList.Announcements = dbPl.Announcements
				Compare2Playlists(pl, dbPlayList)
			}
		}

		if !playListFound {
			var pmiPlayList PmiPlayList
			pmiPlayList.Id = pl.Id
			pmiPlayList.Title = pl.Title
			pmiPlayList.Announcements = pl.Announcements
			Db.Create(&pmiPlayList)
		}
	}

}
