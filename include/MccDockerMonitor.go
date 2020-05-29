package include

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"io/ioutil"
	"log"
	"net/http"
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

		var recs []PodStat
		Db.Find(&recs)
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
