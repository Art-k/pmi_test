package include

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"

	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type PodStatNumber struct {
	gorm.Model
	StatNumber string
}

type PodReplicas struct {
	gorm.Model
	StatNumber  string
	PodName     string
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
	StatNumber string
	PodName    string `json:"pod_name"`
	CPU        int    `json:"cpu"`
	CPUUnit    string `json:"cpu_unit"`
	//PodReplicas int    `json:"replica_count"`
}

type PodCpuMaxByHour struct {
	gorm.Model
	Year       int    `json:"year"`
	Month      int    `json:"month"`
	Day        int    `json:"day"`
	Hour       int    `json:"hour"`
	StatNumber string `json:"stat_code"`
	PodName    string `json:"pod_name"`
	CPU        int    `json:"cpu"`
	CPUUnit    string `json:"cpu_unit"`
	//PodReplicas int    `json:"replica_count"`
}

type PodRamMax struct {
	gorm.Model
	StatNumber string
	PodName    string `json:"pod_name"`
	RAM        int    `json:"ram"`
	RAMUnit    string `json:"ram_unit"`
	//PodReplicas int    `json:"replica_count"`
}

type PodReplicaMax struct {
	gorm.Model
	StatNumber  string
	PodName     string `json:"pod_name"`
	PodReplicas int    `json:"replica_count"`
}

type PodStat struct {
	gorm.Model
	StatNumber string
	PostPodStat
}

func APIMccDockerMonitorReplicaMax(w http.ResponseWriter, r *http.Request) {

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
	case "DELETE":

	default:
		ResponseUnknown(w, "Method is not allowed")
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
	case "DELETE":

	default:
		ResponseUnknown(w, "Method is not allowed")
	}

}

func APIMccDockerMonitorReplicas(w http.ResponseWriter, r *http.Request) {

	switch r.Method {

	case "GET":

		start_data := r.URL.Query().Get("start-data")
		end_data := r.URL.Query().Get("end-data")
		page := r.URL.Query().Get("page")
		per_page := r.URL.Query().Get("per-page")
		pod_name := r.URL.Query().Get("pod-name")
		pod_code := r.URL.Query().Get("pod-code")

		var recs []PodReplicas

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

		//Db.Find(&recs)

		response, err := json.Marshal(&recs)
		if err != nil {
			ResponseBadRequest(w, err, "")
		}
		ResponseOK(w, response)

	case "DELETE":

		scope := r.URL.Query().Get("scope")

		if scope != "" {
			if scope == "ALL" {

				Db.Unscoped().Delete(&PodReplicas{})
				ResponseOK(w, []byte("all records are Deleted"))

				return
			}
		}

		ResponseBadRequest(w, nil, "Please define Scope")
		return

	default:
		ResponseUnknown(w, "Method is not allowed")
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

			Db.Where("stat_number = ?", statNumber.StatNumber).
				Where("pod_name = ?", stat.PodName).
				Find(&repl)

			if repl.ID == 0 {
				repl.StatNumber = statNumber.StatNumber
				repl.PodReplicas = 1
				repl.PodName = stat.PodName
				Db.Create(&repl)
			} else {
				repl.PodReplicas += 1
				Db.Save(&repl)
			}

			var cpuMax PodCpuMax
			Db.Where("pod_name = ?", rec.PodName).Order("created_at desc").Limit(1).Find(&cpuMax)
			if cpuMax.ID == 0 {
				cpuMax.StatNumber = statNumber.StatNumber
				cpuMax.PodName = rec.PodName
				cpuMax.CPU = stat.CPU
				cpuMax.CPUUnit = stat.CPUUnit
				Db.Create(&cpuMax)
			} else {
				if cpuMax.CPU < stat.CPU {
					var newMax PodCpuMax
					newMax.StatNumber = statNumber.StatNumber
					newMax.PodName = rec.PodName
					newMax.CPU = stat.CPU
					newMax.CPUUnit = stat.CPUUnit
					Db.Create(&newMax)
					PostTelegrammMessage("New CPU Load Maximum is Reached, POD : '" + rec.PodName + "'" + ", value : " + strconv.Itoa(stat.CPU) + " (" + statNumber.StatNumber + ")")
				}
			}

			var cpuMaxHour PodCpuMaxByHour
			Hour := time.Now().Hour()
			Year := time.Now().Year()
			Month := int(time.Now().Month())
			Day := time.Now().Day()

			Db.Where("pod_name = ?", rec.PodName).
				Where("hour = ?", Hour).
				Where("year = ?", Year).
				Where("day = ?", Day).
				Where("month = ?", Month).
				Order("created_at desc").Limit(1).Find(&cpuMaxHour)
			if cpuMaxHour.ID == 0 {
				cpuMaxHour.Hour = Hour
				cpuMaxHour.Year = Year
				cpuMaxHour.Day = Day
				cpuMaxHour.Month = Month
				cpuMaxHour.StatNumber = statNumber.StatNumber
				cpuMaxHour.PodName = rec.PodName
				cpuMaxHour.CPU = stat.CPU
				cpuMaxHour.CPUUnit = stat.CPUUnit
				Db.Create(&cpuMaxHour)
			} else {
				if cpuMax.CPU < stat.CPU {
					var newMax PodCpuMaxByHour
					newMax.Hour = Hour
					newMax.Year = Year
					newMax.Day = Day
					newMax.Month = Month
					newMax.StatNumber = statNumber.StatNumber
					newMax.PodName = rec.PodName
					newMax.CPU = stat.CPU
					newMax.CPUUnit = stat.CPUUnit
					Db.Create(&newMax)
					//PostTelegrammMessage("New CPU Load Maximum is Reached, POD : '" + rec.PodName + "'" + ", value : " + strconv.Itoa(stat.CPU) + " (" + statNumber.StatNumber + ")")
				}
			}

			var ramMax PodRamMax
			Db.Where("pod_name = ?", rec.PodName).Order("created_at desc").Limit(1).Find(&ramMax)
			if ramMax.ID == 0 {
				ramMax.StatNumber = statNumber.StatNumber
				ramMax.PodName = rec.PodName
				ramMax.RAM = stat.RAM
				ramMax.RAMUnit = stat.RAMUnit
				Db.Create(&ramMax)
			} else {
				if ramMax.RAM < stat.RAM {
					var newMax PodRamMax
					newMax.StatNumber = statNumber.StatNumber
					newMax.PodName = rec.PodName
					newMax.RAM = stat.RAM
					newMax.RAMUnit = stat.RAMUnit
					Db.Create(&newMax)
					//PostTelegrammMessage("New RAM Load Maximum is Reached, POD : '" + rec.PodName + "'" + ", value : " + strconv.Itoa(stat.RAM) + " (" + statNumber.StatNumber + ")")
				}
			}
		}

		ResponseOK(w, []byte("added"))

	default:
		ResponseUnknown(w, "Method is not allowed")
	}

}
