package include

import (
	"encoding/json"
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
	currentStatus  string
}

type CopiedNoticesHistory struct {
	gorm.Model
	DestinationPlayListsId uint
	Status                 string
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
			}
			PostTelegrammMessage(ID + " task DELETE is completed successfully")
		}(params["id"], notices)

		response, _ := json.Marshal(notices)
		ResponseOK(w, response)
	}
}

func GetUsedCopy(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		var response []string
		var statuses []string
		statuses = append(statuses, "active")
		statuses = append(statuses, "future")

		//cache, _ = bigcache.NewBigCache(bigcache.DefaultConfig(30 * time.Minute))

		U := os.Getenv("USER")
		P := os.Getenv("PASSWORD")

		AllPlaylists := GetAllPlaylists(U, P)

		var allCopyTasks []CopyNoticesToPlaylistsTask
		Db.Find(&allCopyTasks)

		for _, task := range allCopyTasks {
			log.Println("Task", task.ID)
			var copies []DestinationPlaylists
			Db.Where("task_id = ?", task.ID).Order("playlist_id asc").Find(&copies)
			for _, copiedEl := range copies {
				if copiedEl.IsDeleted {
					continue
				}

				currentNotice := GetNoticeFromPlaylistById(copiedEl.PlaylistId, copiedEl.NoticesId, statuses, U, P)
				if currentNotice.Id == 0 {
					log.Println("Notice not Found in expected statuses")
					continue
				}
				var copyNotice TypeNotice
				err := json.Unmarshal([]byte(copiedEl.Notice), &copyNotice)
				if err != nil {
					log.Println(err)
					continue
				}
				if currentNotice.Status != copiedEl.currentStatus {

					var pl_name string

					for _, pl := range AllPlaylists {
						if pl.Id == copiedEl.PlaylistId {
							pl_name = pl.Title
						}
					}

					copiedEl.currentStatus = currentNotice.Status
					Db.Model(&DestinationPlaylists{}).Update(&copiedEl)
					var history CopiedNoticesHistory
					history.DestinationPlayListsId = copiedEl.ID
					history.Status = copiedEl.currentStatus
					Db.Create(&history)

					msg := "Playlist '" + pl_name + "' (" + strconv.Itoa(copiedEl.PlaylistId) + "), notice '" + copyNotice.Title + "' has new status " + currentNotice.Status
					log.Println(msg)
					response = append(response, msg)

				}
			}
		}
		resp, _ := json.Marshal(response)
		ResponseOK(w, resp)
	}
}

func CopyNotes(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":

		var tasks []CopyNoticesToPlaylistsTask
		Db.Order("created_at desc").Find(&tasks)
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
