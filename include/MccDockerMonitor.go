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
	PodName string `json:"pod_name"`
	PodCode string `json:"pod_code"`
	CPU     int    `json:"cpu"`
	CPUUnit string `json:"cpu_unit"`
	RAM     int    `json:"ram"`
	RAMUnit string `json:"ram_unit"`
}

type PodCpuMax struct {
	gorm.Model
	PodName     string `json:"pod_name"`
	CPU         int    `json:"cpu"`
	CPUUnit     string `json:"cpu_unit"`
	PodReplicas int    `json:"replica_count"`
}

type PodRamMax struct {
	gorm.Model
	PodName     string `json:"pod_name"`
	RAM         int    `json:"cpu"`
	RAMUnit     string `json:"ram_unit"`
	PodReplicas int    `json:"replica_count"`
}

type PodStat struct {
	gorm.Model
	StatNumber string
	PostPodStat
}

func APIMccDockerMonitorCpuMax(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		pod_name := r.URL.Query().Get("pod_name")

		DB := Db

		if pod_name != "" {
			DB = DB.Where("pod_name = ?", pod_name)
		}

		var recs []PodCpuMax
		DB.Order("created_at desc").Find(&recs)
		response, err := json.Marshal(&recs)
		if err != nil {
			ResponseBadRequest(w, err, "")
		}
		ResponseOK(w, response)

	}
}

func APIMccDockerMonitorRamMax(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		pod_name := r.URL.Query().Get("pod_name")

		DB := Db

		if pod_name != "" {
			DB = DB.Where("pod_name = ?", pod_name)
		}

		var recs []PodRamMax
		DB.Order("created_at desc").Find(&recs)
		response, err := json.Marshal(&recs)
		if err != nil {
			ResponseBadRequest(w, err, "")
		}
		ResponseOK(w, response)

	}

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

			var cpuMax PodCpuMax
			Db.Where("pod_name = ?", rec.PodName).Order("created_at desc").Limit(1).Find(&cpuMax)
			if cpuMax.ID == 0 {
				cpuMax.PodName = rec.PodName
				cpuMax.CPU = stat.CPU
				cpuMax.CPUUnit = stat.CPUUnit
				Db.Create(&cpuMax)
			} else {
				if cpuMax.CPU < stat.CPU {
					var newMax PodCpuMax
					newMax.PodName = rec.PodName
					newMax.CPU = stat.CPU
					newMax.CPUUnit = stat.CPUUnit
					Db.Create(&newMax)
					//PostTelegrammMessage("New CPU Load Maximum is Reached, POD : '" + rec.PodName + "'" + "value : " + strconv.Itoa(stat.CPU) + " (" + statNumber.StatNumber + ")")
				}
			}

			var ramMax PodRamMax
			Db.Where("pod_name = ?", rec.PodName).Order("created_at desc").Limit(1).Find(&ramMax)
			if ramMax.ID == 0 {
				ramMax.PodName = rec.PodName
				ramMax.RAM = stat.CPU
				ramMax.RAMUnit = stat.CPUUnit
				Db.Create(&ramMax)
			} else {
				if ramMax.RAM < stat.RAM {
					var newMax PodRamMax
					newMax.PodName = rec.PodName
					newMax.RAM = stat.CPU
					newMax.RAMUnit = stat.CPUUnit
					Db.Create(&newMax)
					//PostTelegrammMessage("New RAM Load Maximum is Reached, POD : '" + rec.PodName + "'" + "value : " + strconv.Itoa(stat.RAM) + " (" + statNumber.StatNumber + ")")
				}
			}
		}

		ResponseOK(w, []byte("added"))

	default:
		ResponseUnknown(w, "Method is not allowed")
	}

}
