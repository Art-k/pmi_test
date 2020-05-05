package include

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"net/http"
	"time"
)

type Test struct {
	gorm.Model
	Type             string
	Status           string
	Description      string
	Duration         int
	ErrorCount       int
	RunType          string
	PlayListsTested  int
	PlayListsIgnored int
	Hash             string
}

type TestError struct {
	gorm.Model
	TestId      uint
	Type        string
	Message     string
	Description string
}

type IgnoredPlaylist struct {
	gorm.Model
	PlayListId int
}

func GetIgnoredPlaylists() []int {
	var res []int
	var ignoredPlayLists []IgnoredPlaylist
	Db.Find(&ignoredPlayLists)
	for _, el := range ignoredPlayLists {
		res = append(res, el.PlayListId)
	}
	return res
}

func GetTestsStatistics(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		var tests []Test
		Db.Find(&tests)
		response, _ := json.Marshal(tests)
		ResponseOK(w, response)
	case "POST":
		if !NoticeInJsonTestIsRunning {
			go DoNoticesInJsonTest("Run Over HTTP")
		}
		time.Sleep(1 * time.Second)
		var test Test
		Db.Last(&test)
		response, _ := json.Marshal(test)
		ResponseOK(w, response)
	}
}

func IgnoredPlayLists(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":

		type IncomingData struct {
			PlayListId int
		}

		var incomingData IncomingData
		err := json.NewDecoder(r.Body).Decode(&incomingData)
		if err != nil {
			ResponseBadRequest(w, err, "")
			return
		}
		var rec IgnoredPlaylist
		Db.Where("play_list_id = ?", incomingData.PlayListId).Delete(&rec)

		response, _ := json.Marshal(rec)
		ResponseOK(w, response)

	case "GET":

		var rec []IgnoredPlaylist
		Db.Find(&rec)
		response, _ := json.Marshal(rec)
		ResponseOK(w, response)

	case "POST":

		type IncomingData struct {
			PlayListId int
		}

		var incomingData IncomingData
		err := json.NewDecoder(r.Body).Decode(&incomingData)
		if err != nil {
			ResponseBadRequest(w, err, "")
			return
		}
		var rec IgnoredPlaylist
		Db.Where("play_list_id = ?", incomingData.PlayListId).First(&rec)
		if rec.ID == 0 {
			rec.PlayListId = incomingData.PlayListId
			Db.Create(&rec)
		}
		response, _ := json.Marshal(rec)
		ResponseOK(w, response)

	}
}

func GetTestStatistics(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	switch r.Method {
	case "GET":
		var testerrors []TestError
		Db.Where("test_id = ?", params["id"]).Find(&testerrors)
		response, _ := json.Marshal(testerrors)
		ResponseOK(w, response)
	}
}
