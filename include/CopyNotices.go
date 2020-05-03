package include

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"log"
	"net/http"
	"os"
	"time"
)

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
	TaskId     uint
	PlaylistId int
	NoticesId  int
	Notice     string
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
		Db.Where("task_id = ?", params["id"]).Find(&destination)
		response, _ := json.Marshal(destination)
		ResponseOK(w, response)

	case "DELETE":
		var notices []DestinationPlaylists

		U := os.Getenv("USER")
		P := os.Getenv("PASSWORD")

		Db.Where("task_id = ?", params["id"]).Find(&notices)
		go func() {
			for _, notice := range notices {
				err := DeleteNoticeById(notice.NoticesId, U, P)
				if err != nil {
					log.Println(err)
				} else {
					Db.Where("id = ?", notice.ID).Delete(DestinationPlaylists{})
					var rec CopyNoticesToPlaylistsTask
					Db.Where("id = ?", notice.TaskId).Find(&rec)
					rec.Deleted++
					Db.Model(&CopyNoticesToPlaylistsTask{}).Update(rec)
					time.Sleep(1 * time.Second)
				}
			}
		}()

		response, _ := json.Marshal(notices)
		ResponseOK(w, response)
	}
}

func CopyNotes(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":

		var tasks []CopyNoticesToPlaylistsTask
		Db.Find(&tasks)
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
		go func() {
			for _, pl_id := range incomingData.DestinationPlayLists {
				var destination DestinationPlaylists
				destination.TaskId = incTask.ID
				destination.PlaylistId = pl_id
				CopiedNotice := PostNoticesToPlaylist(SourceNotice, U, P)
				destination.NoticesId = CopiedNotice.Id
				Db.Create(&destination)
				if incTask.CopySchedule {
					CopiedNotice = AssignPlaylists(pl_id, CopiedNotice.Id, SourceNotice.Schedule.Duration,
						SourceNotice.Schedule.ActivateFrom, SourceNotice.Schedule.ActivateTo, U, P)
				} else {
					CopiedNotice = AssignPlaylists(pl_id, CopiedNotice.Id, SourceNotice.Schedule.Duration, incTask.ActivateFrom, incTask.ActivateTo, U, P)
				}
				cn_str, _ := json.Marshal(CopiedNotice)
				destination.Notice = string(cn_str)
				Db.Model(&DestinationPlaylists{}).Update(&destination)
				incTask.Copied++
				time.Sleep(2 * time.Second)
				Db.Model(&CopyNoticesToPlaylistsTask{}).Update(incTask)
			}

			incTask.Status = "Completed"
			incTask.Duration = int(time.Since(start).Seconds())
			Db.Model(&CopyNoticesToPlaylistsTask{}).Update(incTask)
		}()
		response, _ := json.Marshal(incTask)
		ResponseOK(w, response)
	}
}
