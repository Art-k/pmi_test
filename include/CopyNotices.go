package include

import (
	"encoding/json"
	"fmt"
	"github.com/allegro/bigcache"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var cache *bigcache.BigCache

type CopyNoticesToPlaylists struct {
	SourcePlaylistId     int
	SourceNoticeId       int
	DestinationPlayLists []int
	ActivateFrom         string
	ActivateTo           string
	CopySchedule         bool
}

type CopyNoticesToPlaylistsTask struct {
	gorm.Model
	SourcePlaylistId int
	SourceNoticeId   int
	ActivateFrom     string
	ActivateTo       string
	CopySchedule     bool
	SourceNotice     string
	Status           string
	Duration         int
	Copied           int
	Deleted          int
}

type DestinationPlaylists struct {
	gorm.Model
	TaskId         uint
	PlaylistId     int
	NoticesId      int
	Notice         string
	DeletedMessage string
	IsDeleted      bool
	CurrentStatus  string
}

type CopiedNoticesHistory struct {
	gorm.Model
	DestinationPlayListsId uint
	Status                 string
}

func GetAllNoticesGroupBy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case "GET":

		AllPlaylists := GetAllPlaylists(os.Getenv("USER"), os.Getenv("PASSWORD"))

		rows, err := Db.Table("destination_playlists").Select("playlist_id as Id, count(id) as Count").Group("playlist_id").Rows()
		defer rows.Close()
		if err != nil {
			fmt.Println(err)
		}

		type PlaylistStat struct {
			PlaylistID   int
			PlayListName string
			CountCopies  int
		}

		var PlaylistStatResponse []PlaylistStat

		for rows.Next() {
			var Rec PlaylistStat
			err := rows.Scan(&Rec.PlaylistID, &Rec.CountCopies)
			if err != nil {
				fmt.Println(err)
			}
			for _, pl := range AllPlaylists {
				if pl.Id == Rec.PlaylistID {
					Rec.PlayListName = pl.Title
				}
			}

			PlaylistStatResponse = append(PlaylistStatResponse, Rec)
		}

		response, _ := json.Marshal(&PlaylistStatResponse)

		ResponseOK(w, response)

	}
}

func GetAllPlaylistsAsArrayOfId(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		U := os.Getenv("USER")
		P := os.Getenv("PASSWORD")
		Playlists := GetAllPlaylists(U, P)
		var PlaylistsResponse []int
		for _, pl := range Playlists {
			PlaylistsResponse = append(PlaylistsResponse, pl.Id)
		}
		response, _ := json.Marshal(PlaylistsResponse)
		ResponseOK(w, response)
	}
}

func CopyNotesTask(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	switch r.Method {

	case "GET":
		var destination []DestinationPlaylists
		Db.Where("task_id = ?", params["id"]).Order("notices_id asc").Find(&destination)

		type OneTaskResponse struct {
			Total   int
			Deleted int
			Records []DestinationPlaylists
		}

		var oneTaskResponse OneTaskResponse
		oneTaskResponse.Total = len(destination)
		oneTaskResponse.Records = destination
		for _, rec := range destination {
			if rec.IsDeleted {
				oneTaskResponse.Deleted++
			}
		}

		response, _ := json.Marshal(oneTaskResponse)
		ResponseOK(w, response)

	case "DELETE":

		var notices []DestinationPlaylists

		U := os.Getenv("USER")
		P := os.Getenv("PASSWORD")

		Db.Where("task_id = ?", params["id"]).Order("notices_id asc").Find(&notices)

		go func(ID string, list []DestinationPlaylists) {
			PostTelegrammMessage(ID + " task DELETE is started")
			for _, notice := range list {

				currentPlaylistNotice := GetNoticeById(notice.NoticesId, U, P)
				if currentPlaylistNotice.Status == "expired" {
					msg, err := DeleteNoticeById(notice.NoticesId, U, P)
					if err != nil {
						log.Println(err)
					} else {
						notice.DeletedMessage = msg
						notice.IsDeleted = true
						Db.Model(&DestinationPlaylists{}).Where("id = ?", notice.ID).Update(&notice)
						//Db.Where("id = ?", notice.ID).Delete(DestinationPlaylists{})
						var rec CopyNoticesToPlaylistsTask
						Db.Where("id = ?", notice.TaskId).Find(&rec)
						rec.Deleted++
						Db.Model(&CopyNoticesToPlaylistsTask{}).Update(rec)
						time.Sleep(250 * time.Millisecond)
					}
				} else {
					log.Println("ERROR -> NOTICE ID :", notice.NoticesId, " can't delete, status is changed")
				}
			}
			PostTelegrammMessage(ID + " task DELETE is completed successfully")
		}(params["id"], notices)

		response, _ := json.Marshal(notices)
		ResponseOK(w, response)
	}
}

func GetComparesTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		var comparesTasks []ComparesTaskType
		Db.Find(&comparesTasks)

		resp, _ := json.Marshal(comparesTasks)
		ResponseOK(w, resp)
	}
}

type CheckedNotices struct {
	CheckedID int `gorm:"unique"`
}

func ifInArray(index int, array []int) bool {
	for _, ind := range array {
		if ind == index {
			return true
		}
	}
	return false
}

func FixPlaylists(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":

		var CheckedDB []CheckedNotices
		Db.Find(&CheckedDB)
		var Checked []int
		for _, ind := range CheckedDB {
			Checked = append(Checked, ind.CheckedID)
		}

		go func() {

			AllPlaylists := GetAllPlaylists(os.Getenv("USER"), os.Getenv("PASSWORD"))

			for ind, playlist := range AllPlaylists {

				fmt.Println("\n", ind, playlist, len(AllPlaylists))
				plId := playlist.Id

				//go func(plId int) {

				AllNotices := GetAllNoticesByPlaylist(plId, os.Getenv("USER"), os.Getenv("PASSWORD"))

				for indN, notice := range AllNotices {

					if ifInArray(notice.Id, Checked) {
						fmt.Print("+")
						continue
					}

					if notice.CreatedAt == "0000-00-00 00:00:00" {
						fmt.Print("-")
						continue
					}

					date := "2020-05-01"
					t, _ := time.Parse("2006-01-02", date)

					t1, err := time.Parse("2006-01-02 15:04:05", notice.CreatedAt)
					if err != nil {
						fmt.Println(err)
					}

					//fmt.Println(t.Format("Jan 2, 2006 at 3:04pm (MST)"),"|", t1.Format("Jan 2, 2006 at 3:04pm (MST)"))

					if t1.Before(t) {
						fmt.Print("-")
						continue
					}

					fmt.Print("+")

					var destinationPl []DestinationPlaylists

					Db.Where("notices_id = ?", notice.Id).Find(&destinationPl)

					for _, dp := range destinationPl {

						if dp.PlaylistId != plId {
							was := dp.PlaylistId
							dp.PlaylistId = plId
							fmt.Println("\n\t\tindex", indN, "notice ID", notice.Id, "copy pl id", dp.ID, "WAS", was, "NOW", dp.PlaylistId)

							Db.Model(DestinationPlaylists{}).Where("ID = ?", dp.ID).Update("playlist_id", plId)

							var checked CheckedNotices
							checked.CheckedID = notice.Id
							Db.Create(&checked)

						} else {

							var checked CheckedNotices
							checked.CheckedID = notice.Id
							Db.Create(&checked)

						}

					}

				}
				//}(playlist.Id)
			}
		}()

		ResponseOK(w, []byte(""))

	}
}

func GetActiveCopy(w http.ResponseWriter, r *http.Request) {
	//start := time.Now()

	//var task ComparesTaskType
	//task.TaskType = task_type
	switch r.Method {
	case "GET":
		playLists := GetAllPlaylists(os.Getenv("USER"), os.Getenv("PASSWORD"))

		var currentPlaylist TypePlaylist
		var notices []TypeNotice
		var responseStr []string
		var copies []DestinationPlaylists

		Db.Where("is_deleted = ?", false).Order("playlist_id asc").Find(&copies)
		for _, copiedEl := range copies {
			if currentPlaylist.Id != copiedEl.PlaylistId {
				fmt.Println("#### GET PLAYLIST ####", copiedEl.PlaylistId)
				notices = GetAllNoticesByPlaylist(copiedEl.PlaylistId, os.Getenv("USER"), os.Getenv("PASSWORD"))
				//currentPlaylist = copiedEl.PlaylistId
				for _, playlist := range playLists {
					if copiedEl.PlaylistId == playlist.Id {
						currentPlaylist = playlist
						break
					}
				}
			}

			for _, notice := range notices {
				if notice.Id == copiedEl.NoticesId {
					if copiedEl.CurrentStatus == "active" {
						Msg := "Playlist '" + currentPlaylist.Title + "' (" + strconv.Itoa(currentPlaylist.Id) + ") , Notice '" + notice.Title + "' has status 'ACTIVE'"
						fmt.Println(Msg)
						responseStr = append(responseStr, Msg)
					}
				}
			}
		}
		responseStr = append(responseStr, "Total :"+strconv.Itoa(len(responseStr)))
		resp, _ := json.Marshal(responseStr)
		ResponseOK(w, resp)

	}
}

func GetUsedCopy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		go func() {
			CompareStatusesCopiedNotices("Call from API")
		}()
		ResponseOK(w, nil)
	case "GET":

		//background := r.URL.Query().Get("background")

		type historyResponseType struct {
			Notice  DestinationPlaylists
			History []CopiedNoticesHistory
		}

		var historyResponses []historyResponseType

		var copies []DestinationPlaylists
		Db.Where("is_deleted = ?", false).Find(&copies)

		for _, copiedEl := range copies {
			var history []CopiedNoticesHistory
			Db.Where("destination_play_lists_id = ?", copiedEl.ID).Find(&history)
			if len(history) > 1 {
				var historyResponse historyResponseType
				historyResponse.Notice = copiedEl
				historyResponse.History = history

				historyResponses = append(historyResponses, historyResponse)

			}
		}

		resp, _ := json.Marshal(historyResponses)
		ResponseOK(w, resp)

	}
}

var DoingHistory bool

func MakeHistory(t time.Time) {
	if !DoingHistory {
		CompareStatusesCopiedNotices("auto")
	}
}

type ComparesTaskType struct {
	gorm.Model
	TaskType string
	Changes  int
	Duration int
}

func CompareStatusesCopiedNotices(task_type string) {

	DoingHistory = true

	start := time.Now()

	var task ComparesTaskType
	task.TaskType = task_type
	Db.Create(&task)
	var currentPlaylist int
	var notices []TypeNotice

	var copies []DestinationPlaylists
	Db.Where("is_deleted = ?", false).Order("playlist_id asc").Find(&copies)
	for _, copiedEl := range copies {
		if currentPlaylist != copiedEl.PlaylistId {
			log.Println("#### GET PLAYLIST ####", copiedEl.PlaylistId)
			notices = GetAllNoticesByPlaylist(copiedEl.PlaylistId, os.Getenv("USER"), os.Getenv("PASSWORD"))
			currentPlaylist = copiedEl.PlaylistId
		}
		for _, notice := range notices {
			if notice.Id == copiedEl.NoticesId {
				if copiedEl.CurrentStatus != notice.Status {
					task.Changes++
					copiedEl.CurrentStatus = notice.Status
					Db.Model(&DestinationPlaylists{}).Update(copiedEl)

					var his CopiedNoticesHistory
					his.DestinationPlayListsId = copiedEl.ID
					his.Status = notice.Status
					Db.Create(&his)

					log.Println("Status is changed to ", notice.Status)

				} else {
					log.Println("The same Status")
				}
			}
		}
	}

	task.Duration = int(time.Since(start).Seconds())

	Db.Model(&ComparesTaskType{}).Update(&task)
	PostTelegrammMessage("Notices History Created " + strconv.Itoa(task.Changes) + " changes found. Task id = " + strconv.Itoa(int(task.ID)))
	DoingHistory = false
}

func FuncHistory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		var history []CopiedNoticesHistory
		Db.Find(&history)
		histories := map[string]interface{}{
			"Records": history,
			"Total":   len(history),
		}
		response, _ := json.Marshal(histories)
		ResponseOK(w, response)
		return

	case "DELETE":
		params := mux.Vars(r)
		if params["id"] != "ALL" {

			Db.Where("id = ?", params["id"]).Delete(CopiedNoticesHistory{})
			ResponseOK(w, nil)
			return

		} else {

			Db.Unscoped().Delete(CopiedNoticesHistory{})
			ResponseOK(w, nil)
			return

		}
	}
}

func CopyNotes(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":

		var page int
		var perPage int

		pageStr := r.URL.Query().Get("page")

		if pageStr == "" {
			page = 1
		} else {
			page, _ = strconv.Atoi(pageStr)
		}

		perPageStr := r.URL.Query().Get("per-page")
		if perPageStr == "" {
			perPage = 15
		} else {
			perPage, _ = strconv.Atoi(perPageStr)
			if perPage > 1000 {
				perPage = 1000
			}
		}

		var tasks []CopyNoticesToPlaylistsTask
		Db.Order("created_at desc").Offset((page - 1) * perPage).Limit(perPage).Find(&tasks)
		response, _ := json.Marshal(tasks)
		ResponseOK(w, response)

	case "POST":

		start := time.Now()

		var incomingData CopyNoticesToPlaylists
		err := json.NewDecoder(r.Body).Decode(&incomingData)
		if err != nil {
			ResponseBadRequest(w, err, "")
			return
		}

		var SourceNotice TypeNotice

		var incTask CopyNoticesToPlaylistsTask
		incTask.CopySchedule = incomingData.CopySchedule
		incTask.ActivateTo = incomingData.ActivateTo
		incTask.ActivateFrom = incomingData.ActivateFrom
		incTask.SourcePlaylistId = incomingData.SourcePlaylistId
		incTask.SourceNoticeId = incomingData.SourceNoticeId
		incTask.SourceNotice = ""

		U := os.Getenv("USER")
		P := os.Getenv("PASSWORD")

		AllNotices := GetAllNoticesByPlaylist(incTask.SourcePlaylistId, U, P)

		for _, notice := range AllNotices {
			if notice.Id == incTask.SourceNoticeId {
				notice_str, err := json.Marshal(&notice)
				if err != nil {
					incTask.Status = "Error, source notice can't be saved as a string"
					ResponseBadRequest(w, err, incTask.Status)
				}
				incTask.SourceNotice = string(notice_str)
				SourceNotice = notice
				break
			}
		}

		Db.Create(&incTask)
		if incTask.Status != "" {
			ResponseBadRequest(w, nil, incTask.Status)
		}

		go func(Source_Notice TypeNotice, Dplaylists []int) {

			PostTelegrammMessage(strconv.Itoa(int(incTask.ID)) + " task COPY is started")

			for _, pl_id := range Dplaylists {
				var destination DestinationPlaylists
				destination.TaskId = incTask.ID
				destination.PlaylistId = pl_id
				CopiedNotice := PostNoticesToPlaylist(Source_Notice, U, P)
				cn_str, _ := json.Marshal(CopiedNotice)
				destination.NoticesId = CopiedNotice.Id
				Db.Create(&destination)
				if incTask.CopySchedule {
					CopiedNotice = AssignPlaylists(pl_id, CopiedNotice.Id, Source_Notice.Schedule.Duration,
						Source_Notice.Schedule.ActivateFrom, Source_Notice.Schedule.ActivateTo, U, P)
				} else {
					CopiedNotice = AssignPlaylists(pl_id, CopiedNotice.Id, Source_Notice.Schedule.Duration, incTask.ActivateFrom, incTask.ActivateTo, U, P)
				}
				destination.Notice = string(cn_str)
				Db.Model(&DestinationPlaylists{}).Update(&destination)
				incTask.Copied++
				time.Sleep(2 * time.Second)
				Db.Model(&CopyNoticesToPlaylistsTask{}).Update(&incTask)
			}
			incTask.Status = "Copy Completed"
			incTask.Duration = int(time.Since(start).Seconds())
			Db.Model(&CopyNoticesToPlaylistsTask{}).Update(&incTask)
			PostTelegrammMessage(strconv.Itoa(int(incTask.ID)) + " task COPY is completed successfully")

		}(SourceNotice, incomingData.DestinationPlayLists)

		response, _ := json.Marshal(incTask)
		ResponseOK(w, response)
	}
}
