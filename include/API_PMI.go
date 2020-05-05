package include

import (
	"bytes"
	"encoding/json"
	"github.com/jinzhu/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Token struct {
	gorm.Model
	Token       string
	Host        string
	ServiceName string
	User        string
}

type TypeToken struct {
	Executems int    `json:"executems"`
	Token     string `json:"token"`
}

type TypeService struct {
	Executems  int       `json:"executems"`
	Host       string    `json:"host"`
	Id         uint      `json:"id"`
	Name       string    `json:"name"`
	Response   TypeToken `json:"response"`
	Status     string    `json:"status"`
	StatusCode uint      `json:"statusCode"`
}

type TypeTokenResponse struct {
	Executems int           `json:"executems"`
	Services  []TypeService `json:"services"`
	Token     string        `json:"token"`
}

func GetPMITokens(user, pass string) {

	requestBody, err := json.Marshal(map[string]string{
		"login":    user,
		"password": pass,
	})

	if err != nil {
		log.Println(err)
	}

	resp, err := http.Post(os.Getenv("AUTH_SERVICE"), "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var tokenResponse TypeTokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		log.Println(err)
	}

	for _, service := range tokenResponse.Services {
		var token Token
		token.User = user
		token.Host = service.Host
		token.ServiceName = service.Name
		token.Token = service.Response.Token
		Db.Create(&token)
	}
}

func GetServiceToken(service, user, pass string) string {
	var token Token
	attempt := 1
	const max_attempt = 5
	for {
		Db.Where("service_name = ?", service).Last(&token)
		if token.ID == 0 {
			GetPMITokens(user, pass)
			time.Sleep(10 * time.Second)
			attempt++
		}
		if attempt > max_attempt {
			return ""
		}
		break
	}
	return token.Token
}

type TypePlaylist struct {
	Id            int    `json:"id"`
	Title         string `json:"title"`
	Announcements int    `json:"announcements"`
}

func GetServiceURL(service string) string {
	var token Token
	Db.Where("service_name = ?", service).Last(&token)
	if token.ID == 0 {
		log.Println("URL for " + service + " is not found")
	}
	return token.Host
}

type TypePlaylistsResponse []TypePlaylist

func GetAllPlaylists(user, pass string) []TypePlaylist {

	client := &http.Client{}

	AnnouncementToken := GetServiceToken("Announcements", user, pass)
	AnnouncementURL := GetServiceURL("Announcements")
	request, err := http.NewRequest("GET", AnnouncementURL+os.Getenv("ANNOUNCEMENT_PLAYLIST"), nil)
	if err != nil {
		log.Println(err)
		return nil
	}
	request.Header.Set("Authorization", "Bearer "+AnnouncementToken)
	resp, err := client.Do(request)
	defer resp.Body.Close()
	if err != nil {
		log.Println(err)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)

	var playlists TypePlaylistsResponse
	err = json.Unmarshal(body, &playlists)
	if err != nil {
		log.Println(err)
		return nil
	}
	return playlists
}

type TypeSchedule struct {
	ActivateFrom    string `json:"activateFrom"`
	ActivateTo      string `json:"activateTo"`
	Duration        int    `json:"duration"`
	Hours           string `json:"hours"`
	Days            string `json:"days"`
	WeekDays        string `json:"weekdays"`
	Month           string `json:"month"`
	Deleted         bool   `json:"deleted"`
	LocalTimeOffset int    `json:"localTimeOffset"`
}

type TypeUser struct {
	Id          uint   `json:"id"`
	Login       string `json:"login"`
	FirstName   string `json:"firstname"`
	LastName    string `json:"lastname"`
	Email       string `json:"email"`
	AccessLevel string `json:"accessLevel"`
	Active      bool   `json:"active"`
}

type TypeNotice struct {
	Id               int          `json:"id"`
	Title            string       `json:"title"`
	Content          string       `json:"content"`
	Pdf              bool         `json:"pdf"`
	CategoryId       uint         `json:"categoryId"`
	BackgroundColor  string       `json:"background_color"`
	BackgroundFileId int64        `json:"background_file_id"`
	CreatedBy        uint         `json:"createdBy"`
	CreatedByUser    TypeUser     `json:"createdByUser"`
	CreatedAt        string       `json:"createdAt"`
	EditedBy         uint         `json:"editedBy"`
	EditedAt         string       `json:"editedAt"`
	Deleted          bool         `json:"deleted"`
	DeletedBy        uint         `json:"deletedBy"`
	DeletedAt        string       `json:"deletedAt"`
	Status           string       `json:"status"`
	Schedule         TypeSchedule `json:"schedule"`
}

type TypePagination struct {
	Found uint `json:"found"`
	Total uint `json:"total"`
}

type TypeNoticesResponse struct {
	Pagination TypePagination `json:"pagination"`
	Response   []TypeNotice   `json:"response"`
}

func GetAllNoticesByPlaylist(playlist_id int, user, pass string) []TypeNotice {

	client := &http.Client{}

	playlist_id_str := strconv.Itoa(playlist_id)

	AnnouncementToken := GetServiceToken("Announcements", user, pass)
	AnnouncementURL := GetServiceURL("Announcements")
	request, err := http.NewRequest("GET", AnnouncementURL+os.Getenv("ANNOUNCEMENT_PLAYLIST")+"/"+playlist_id_str, nil)
	if err != nil {
		log.Println("Error, create requests to get announcements from playlist by id")
		log.Println(err)
		return nil
	}
	request.Header.Set("Authorization", "Bearer "+AnnouncementToken)
	resp, err := client.Do(request)
	defer resp.Body.Close()
	if err != nil {
		log.Println("Error, executing requests to get Announcements from playlist by id")
		log.Println(err)
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)

	var response TypeNoticesResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Error, Unmarshal Response get Announcements from playlist by id")
		log.Println(err)
		return nil
	}
	return response.Response
}

type TypeRepeats struct {
	Days         string `json:"days"`
	DaysForMonth bool   `json:"days_for_month"`
	Hours        string `json:"hours"`
	Month        string `json:"month"`
	Weeks        string `json:"weeks"`
}

type TypeServerNotice struct {
	ActivateEnd   string `json:"activate_end"`
	ActivateStart string `json:"activate_start"`
	AutoScroll    int    `json:"autoscroll"`
	Background    string `json:"background"`
	BackgroundUrl string `json:"background_url"`
	CategoryIcon  string `json:"category_icon"`
	Duration      int    `json:"duration"`
	Emergency     string `json:"emergency"`
	FullPath2File string `json:"fullPath2File"`
	PageId        int    `json:"page_id"`
	Path2File     string `json:"path2file"`
	Path2FileFr   string `json:"path2filefr"`
	Pdf           bool   `json:"pdf"`
	Priority      int    `json:"priority"`
	TypeRepeats   `json:"repeats"`
}

type TypeServiceNoticesResponse struct {
	Entry []TypeServerNotice `json:"entry"`
}

func GetServerPlaylistJson(playlist_id int) []TypeServerNotice {
	playlist_id_str := strconv.Itoa(playlist_id)
	resp, err := http.Get(os.Getenv("MAXTV_NOTICES") + "playlist_new_" + playlist_id_str + "/playlist.json")
	if err != nil {
		log.Println("Error, Get Json File From Server")
		log.Println(err)
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	var response TypeServiceNoticesResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Error, Unmarshal Get Json File From Server")
		log.Println(err)
		return nil
	}

	return response.Entry

}

func PostNoticesToPlaylist(notice TypeNotice, user, pass string) TypeNotice {

	client := &http.Client{}
	var New_Notice TypeNotice

	AnnouncementToken := GetServiceToken("Announcements", user, pass)
	AnnouncementURL := GetServiceURL("Announcements")

	type PostNotice struct {
		Title            string `json:"title"`
		Content          string `json:"content"`
		Pdf              bool   `json:"pdf"`
		CategoryId       uint   `json:"categoryId"`
		BackGroundFileId int64  `json:"background_file_id"`
		BackGroundColor  string `json:"background_color"`
	}

	var newNotice PostNotice
	newNotice.Title = notice.Title
	newNotice.Content = notice.Content
	newNotice.Pdf = notice.Pdf
	newNotice.CategoryId = notice.CategoryId
	newNotice.BackGroundFileId = notice.BackgroundFileId
	newNotice.BackGroundColor = notice.BackgroundColor

	requestBody, err := json.Marshal(newNotice)
	if err != nil {
		return New_Notice
	}
	request, err := http.NewRequest("POST", AnnouncementURL+"/announcement", bytes.NewBuffer(requestBody))
	if err != nil {
		return New_Notice
	}
	request.Header.Set("Authorization", "Bearer "+AnnouncementToken)

	resp, err := client.Do(request)
	if err != nil {
		return New_Notice
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return New_Notice
	}

	log.Println(string(body))
	json.Unmarshal(body, &New_Notice)

	return New_Notice
}

func AssignPlaylists(playlist_id, notice_id, duration int, activate_from, activate_to string, user, pass string) TypeNotice {

	client := &http.Client{}
	AnnouncementToken := GetServiceToken("Announcements", user, pass)
	AnnouncementURL := GetServiceURL("Announcements")
	var New_Notice TypeNotice

	type requestBody struct {
		NoExpire     int    `json:"noExpire"`
		ActivateFrom string `json:"activateFrom"`
		ActivateTo   string `json:"activateTo"`
		Days         string `json:"days"`
		Duration     int    `json:"duration"`
		Hours        string `json:"hours"`
		Month        string `json:"month"`
		PlaylistId   []int  `json:"playlistId"`
		Weekdays     string `json:"weekdays"`
		Expire       int    `json:"expire"`
	}

	var rBody requestBody
	rBody.Duration = duration
	rBody.ActivateTo = activate_to
	rBody.ActivateFrom = activate_from
	rBody.PlaylistId = append(rBody.PlaylistId, playlist_id)
	rBody.Expire = 1

	var r_Body []requestBody
	r_Body = append(r_Body, rBody)

	request_body, err := json.Marshal(r_Body)

	if err != nil {
		log.Println(err)
	}

	request, err := http.NewRequest("POST", AnnouncementURL+"/announcement/"+strconv.Itoa(notice_id)+"/playlist", bytes.NewBuffer(request_body))
	if err != nil {
		return New_Notice
	}
	request.Header.Set("Authorization", "Bearer "+AnnouncementToken)

	resp, _ := client.Do(request)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	log.Println(string(body))
	return New_Notice
}

func DeleteNoticeById(notice_id int, user, pass string) (string, error) {

	client := &http.Client{}
	AnnouncementToken := GetServiceToken("Announcements", user, pass)
	AnnouncementURL := GetServiceURL("Announcements")

	request, err := http.NewRequest("DELETE", AnnouncementURL+"/announcement/"+strconv.Itoa(notice_id)+"/clear", nil)
	if err != nil {
		log.Println(err)
		return "", err
	} else {
		request.Header.Set("Authorization", "Bearer "+AnnouncementToken)
		resp, err := client.Do(request)
		if err != nil {
			log.Println(err)
			return "", err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		log.Println(string(body))
		return string(body), nil
	}
}

func GetNoticeFromPlaylistById(playlist_id, notice_id int, status []string, U, P string) (n TypeNotice) {

	var notices []TypeNotice

	notices = GetAllNoticesByPlaylist(playlist_id, U, P)

	for _, notice := range notices {
		if notice.Id == notice_id {
			return notice
		}
	}

	return n
}
