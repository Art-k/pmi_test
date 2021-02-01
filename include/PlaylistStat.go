package include

import (
	"bytes"
	"encoding/json"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
)

var CompareTaskIsActive bool

type GetPlayListStats struct {
	gorm.Model
	RunType               string
	DurationSec           int
	Status                string
	CMSResponseStatusCode int
	PlayListCount         int
	PlayListProcessed     int
}

type NoticeInPlaylist struct {
	gorm.Model
	PlayListId int
	NoticeId   int
}

type PlayListStat struct {
	PlayListId                int    `json:"playlist_id"`
	PlayListName              string `json:"playlist_name"`
	LastActivity              string `json:"last_activity"`
	NumberOfActiveNotices     int    `json:"active_notices_count"`
	NumberOfExpiredNotices    int    `json:"expired_notices_count"`
	NumberOfFutureNotices     int    `json:"future_notices_count"`
	NumberOfArchivedNotices   int    `json:"archive_notices_count"`
	NumberOfPDFNotices        int    `json:"pdf_notices_count"`
	NumberOfAdvancedScheduled int    `json:"advance_scheduled_count"`
	TotalDurationSeconds      int    `json:"total_duration_sec"`
	AvgActiveDays             int    `json:"avg_post_length"`
	MinActiveDays             int    `json:"min_post_length"`
	MaxActiveDays             int    `json:"max_post_length"`
	NumberOfForeverNotices    int    `json:"forever_count"`
}

type TPlayListStat struct {
	gorm.Model
	TaskID string
	PlayListStat
}

func PostPlayListStatToCMS(task string, stats []PlayListStat) {

	var client *http.Client
	client = &http.Client{}

	jsonStr, _ := json.Marshal(stats)
	req, err := http.NewRequest("POST", os.Getenv("CMS_PLAYLISTS_STAT"), bytes.NewBuffer(jsonStr))

	resp, err := client.Do(req)
	if err != nil {
		log.Println(task, err, "!!! ERROR !!! Post playlist statistic to CMS")
	}
	defer resp.Body.Close()

	var taskStat GetPlayListStats
	Db.Where("id = ?", task).Find(&taskStat)
	taskStat.CMSResponseStatusCode = resp.StatusCode
	Db.Save(&taskStat)

}
