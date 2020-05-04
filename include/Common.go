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

func IfExists(list []int, element int) bool {
	for _, el := range list {
		if el == element {
			return true
		}
	}
	return false
}
