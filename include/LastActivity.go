package include

import (
	"github.com/jinzhu/gorm"
	"time"
)

var GetLastActivityIsRunning bool

type LatestChangesInNotes struct {
	gorm.Model
	PlayListId       uint
	PlayListName     string
	NoticesCount     int
	LastCheck        time.Time
	LastUserActivity time.Time
	ActivityCount    time.Time
}

func GetLastActivity(t time.Time) {
	if !GetLastActivityIsRunning {
		DoGetLastActivity("auto")
	}
}

func DoGetLastActivity(run_type string) {

	NoticeInJsonTestIsRunning = true

}
