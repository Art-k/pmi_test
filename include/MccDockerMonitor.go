package include

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

type PodStatNumber struct {
	gorm.Model
	StatNumber string
}

type PodReplicas struct {
	gorm.Model
	StatNumber  string
	PodName     uint
	PodReplicas int
}

type PostPodStat struct {
	PodName string
	PodCode string
	CPU     int
	CPUUnit string
	RAM     int
	RAMUnit string
}

type PodStat struct {
	gorm.Model
	StatNumber string
	PostPodStat
}

func APIMccDockerMonitor(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case "GET":

		start_data := r.URL.Query().Get("start-data")
		end_data := r.URL.Query().Get("end-data")
		page := r.URL.Query().Get("page")
		per_page := r.URL.Query().Get("per-page")
		pod_name := r.URL.Query().Get("pod-name")
		pod_code := r.URL.Query().Get("pod-code")

		var recs []PodStat

		DB := Db

		if start_data != "" {
			DB = DB.Where("created_at >= ?", start_data)
		}

		if end_data != "" {
			DB = DB.Where("created_at <= ?", end_data)
		}

		if pod_name != "" {
			DB = DB.Where("pod_code = ?", pod_name)
		}

		if pod_code != "" {
			DB = DB.Where("pod_name = ?", pod_code)
		}

		var int_page int
		var int_per_page int

		if per_page != "" {
			int_per_page, _ = strconv.Atoi(per_page)
		} else {
			int_per_page = 10
		}
		fmt.Println("PerPage :", per_page, int_per_page)

		if page == "" {
			int_page = 1
		} else {
			int_page, _ = strconv.Atoi(page)
		}
		if int_page >= 1 {
			fmt.Println(int_page)
			DB = DB.Offset((int_page - 1) * int_per_page)
		}

		DB.Limit(int_per_page).Order("created_at desc").Find(&recs)

		response, err := json.Marshal(&recs)
		if err != nil {
			ResponseBadRequest(w, err, "")
		} else {
			ResponseOK(w, response)
		}

	case "POST":

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
			ResponseBadRequest(w, err, "")
			return
		}

		var recs []PostPodStat
		err = json.Unmarshal(body, &recs)
		if err != nil {
			log.Println(err)
			ResponseBadRequest(w, err, "")
			return
		}

		var statNumber PodStatNumber
		statNumber.StatNumber = GetHash()

		Db.Create(&statNumber)

		for _, stat := range recs {
			var rec PodStat
			rec.StatNumber = statNumber.StatNumber
			rec.PostPodStat = stat
			Db.Create(&rec)

			var repl PodReplicas

			Db.Where("stat_number = ?", statNumber).Find(&repl)

			if repl.ID == 0 {
				repl.StatNumber = statNumber.StatNumber
				repl.PodReplicas = 1
				Db.Create(&repl)
			} else {
				repl.PodReplicas += 1
				Db.Save(&repl)
			}

		}

		ResponseOK(w, []byte("added"))

	default:
		ResponseUnknown(w, "Method is not allowed")
	}

}
