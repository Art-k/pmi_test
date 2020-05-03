package include

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"net/http"
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

func GetTestsStatistics(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		var tests []Test
		Db.Find(&tests)
		response, _ := json.Marshal(tests)
		ResponseOK(w, response)
	case "POST":
		DoNoticesInJsonTest("Run Over HTTP")
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
