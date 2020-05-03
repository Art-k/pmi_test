package include

import (
	"github.com/jinzhu/gorm"
	guuid "github.com/satori/go.uuid"
	"time"
)

var Db *gorm.DB
var Err error

func GetHash() string {
	id, _ := guuid.NewV4()
	return id.String()
}

func DoEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}
