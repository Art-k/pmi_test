package include

import (
	"bytes"
	"encoding/json"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	StatusActive   = "active"
	StatusExpired  = "expired"
	StatusFuture   = "future"
	StatusArchived = "archived"
)

type Token struct {
	gorm.Model
	Token             string
	Host              string
	ServiceName       string
	User              string
	TokenRegistration string
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

	var tokenRegistrationHash string
	tokenRegistrationHash = GetHash()

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
		token.TokenRegistration = tokenRegistrationHash
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
		} else {
			if token.CreatedAt.Before(time.Now().AddDate(0, 0, -14)) {
				GetPMITokens(user, pass)
				time.Sleep(10 * time.Second)
				attempt++
			}
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

	if resp.StatusCode == 401 {
		var token Token
		Db.Where("token = ?", token).Find(&token)
		if token.ID != 0 {
			Db.Delete(&Token{}, "token_registartion = ?", token.TokenRegistration)
		}
		return nil
	}

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
	CreatedByUser    TypeUser     `gorm:"-" ;json:"createdByUser"`
	CreatedAt        string       `json:"createdAt"`
	EditedBy         uint         `json:"editedBy"`
	EditedAt         string       `json:"editedAt"`
	Deleted          bool         `json:"deleted"`
	DeletedBy        uint         `json:"deletedBy"`
	DeletedAt        string       `json:"deletedAt"`
	Status           string       `json:"status"`
	Schedule         TypeSchedule `gorm:"-" ;json:"schedule"`
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

func GetNoticeById(noticeId int, U, P string) (n TypeNotice) {

	var notice TypeNotice

	AnnouncementToken := GetServiceToken("Announcements", U, P)
	AnnouncementURL := GetServiceURL("Announcements")

	client := &http.Client{}

	request, err := http.NewRequest("GET", AnnouncementURL+os.Getenv("ANNOUNCEMENT")+"/"+strconv.Itoa(noticeId), nil)
	if err != nil {
		log.Println("Error, create requests to get announcements from playlist by id")
		log.Println(err)
		return notice
	}
	request.Header.Set("Authorization", "Bearer "+AnnouncementToken)
	resp, err := client.Do(request)
	defer resp.Body.Close()
	if err != nil {
		log.Println("Error, executing requests to get Announcements from playlist by id")
		log.Println(err)
		return notice
	}

	body, err := ioutil.ReadAll(resp.Body)

	var response TypeNotice
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Error, Unmarshal Response get Announcements from playlist by id")
		log.Println(err)
		return notice
	}

	return response
}

func UpdateNoticeById(noticeId int, notice TypeNotice, U, P string) bool {

	AnnouncementToken := GetServiceToken("Announcements", U, P)
	AnnouncementURL := GetServiceURL("Announcements")

	client := &http.Client{}

	requestBody, err := json.Marshal(notice)
	if err != nil {
		return false
	}

	request, err := http.NewRequest("PUT", AnnouncementURL+os.Getenv("ANNOUNCEMENT")+"/"+strconv.Itoa(noticeId), bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("Error, create requests to put Announcement by id")
		log.Println(err)
		return false
	}
	request.Header.Set("Authorization", "Bearer "+AnnouncementToken)
	resp, err := client.Do(request)
	defer resp.Body.Close()
	if err != nil {
		log.Println("Error, executing requests to put Announcement by id")
		log.Println(err)
		return false
	}

	return true
}

type TNoticesDiff struct {
	gorm.Model
	NoticesDiff
}

type NoticesDiff struct {
	RNoticeId        int
	NoticeId         int
	FieldName        string
	RefIntValue      int
	NoticeIntValue   int
	RefStrValue      string
	NoticeStrValue   string
	RefBoolValue     bool
	NoticeBoolValue  bool
	RefUIntValue     uint
	NoticeUIntValue  uint
	RefInt64Value    int64
	NoticeInt64Value int64
}

func AddNoticesDiffToDB(noticesDiff NoticesDiff) {
	Df := TNoticesDiff{
		NoticesDiff: noticesDiff,
	}
	Db.Create(&Df)

	var count int64
	Db.Model(&TNoticesDiff{}).Count(&count)
	if count > 3000000 {
		//needToDeleteCount := count - 2000000
		needToDeleteCount := 1000000
		//var nDiff []TNoticesDiff
		//Db.Model(&TNoticesDiff{}).Order("created_at|ASC").Limit(needToDeleteCount).Count(&count)
		//Db.Order("created_at|ASC").Limit(needToDeleteCount).Unscoped().Delete(&TNoticesDiff{})
		Db.Exec("delete from t_notices_diffs where id IN (SELECT id from t_notices_diffs order by id desc limit " + strconv.Itoa(needToDeleteCount) + ")")
		log.Println("Deleted ", needToDeleteCount, "records")
	}
}

func Compare2Notices(reference, notice TypeNotice) (diff []NoticesDiff, diffLength int) {

	var df NoticesDiff
	df.RNoticeId = reference.Id
	df.NoticeId = notice.Id

	// int
	if reference.Id != notice.Id {
		df.FieldName = "Id"
		df.RefIntValue = reference.Id
		df.NoticeIntValue = notice.Id
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Schedule.Duration != notice.Schedule.Duration {
		df.FieldName = "Schedule.Duration"
		df.RefIntValue = reference.Schedule.Duration
		df.NoticeIntValue = notice.Schedule.Duration
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Schedule.LocalTimeOffset != notice.Schedule.LocalTimeOffset {
		df.FieldName = "Schedule.LocalTimeOffset"
		df.RefIntValue = reference.Schedule.LocalTimeOffset
		df.NoticeIntValue = notice.Schedule.LocalTimeOffset
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}

	// uint
	if reference.CategoryId != notice.CategoryId {
		df.FieldName = "CategoryId"
		df.RefUIntValue = reference.CategoryId
		df.NoticeUIntValue = notice.CategoryId
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.CreatedBy != notice.CreatedBy {
		df.FieldName = "CreatedBy"
		df.RefUIntValue = reference.CreatedBy
		df.NoticeUIntValue = notice.CreatedBy
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.EditedBy != notice.EditedBy {
		df.FieldName = "EditedBy"
		df.RefUIntValue = reference.EditedBy
		df.NoticeUIntValue = notice.EditedBy
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.DeletedBy != notice.DeletedBy {
		df.FieldName = "DeletedBy"
		df.RefUIntValue = reference.DeletedBy
		df.NoticeUIntValue = notice.DeletedBy
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.CreatedByUser.Id != notice.CreatedByUser.Id {
		df.FieldName = "CreatedByUser.Id"
		df.RefUIntValue = reference.CreatedByUser.Id
		df.NoticeUIntValue = notice.CreatedByUser.Id
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}

	// string
	if reference.Title != notice.Title {
		df.FieldName = "Title"
		df.RefStrValue = reference.Title
		df.NoticeStrValue = notice.Title
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Content != notice.Content {
		df.FieldName = "Content"
		df.RefStrValue = reference.Content
		df.NoticeStrValue = notice.Content
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.BackgroundColor != notice.BackgroundColor {
		df.FieldName = "BackgroundColor"
		df.RefStrValue = reference.BackgroundColor
		df.NoticeStrValue = notice.BackgroundColor
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.CreatedAt != notice.CreatedAt {
		df.FieldName = "CreatedAt"
		df.RefStrValue = reference.CreatedAt
		df.NoticeStrValue = notice.CreatedAt
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.EditedAt != notice.EditedAt {
		df.FieldName = "EditedAt"
		df.RefStrValue = reference.EditedAt
		df.NoticeStrValue = notice.EditedAt
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.DeletedAt != notice.DeletedAt {
		df.FieldName = "DeletedAt"
		df.RefStrValue = reference.DeletedAt
		df.NoticeStrValue = notice.DeletedAt
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Status != notice.Status {
		df.FieldName = "Status"
		df.RefStrValue = reference.Status
		df.NoticeStrValue = notice.Status
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}

	if reference.CreatedByUser.Login != notice.CreatedByUser.Login {
		df.FieldName = "CreatedByUser.Login"
		df.RefStrValue = reference.CreatedByUser.Login
		df.NoticeStrValue = notice.CreatedByUser.Login
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.CreatedByUser.AccessLevel != notice.CreatedByUser.AccessLevel {
		df.FieldName = "CreatedByUser.AccessLevel"
		df.RefStrValue = reference.CreatedByUser.AccessLevel
		df.NoticeStrValue = notice.CreatedByUser.AccessLevel
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.CreatedByUser.FirstName != notice.CreatedByUser.FirstName {
		df.FieldName = "CreatedByUser.FirstName"
		df.RefStrValue = reference.CreatedByUser.FirstName
		df.NoticeStrValue = notice.CreatedByUser.FirstName
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.CreatedByUser.LastName != notice.CreatedByUser.LastName {
		df.FieldName = "CreatedByUser.LastName"
		df.RefStrValue = reference.CreatedByUser.LastName
		df.NoticeStrValue = notice.CreatedByUser.LastName
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.CreatedByUser.Email != notice.CreatedByUser.Email {
		df.FieldName = "CreatedByUser.Email"
		df.RefStrValue = reference.CreatedByUser.Email
		df.NoticeStrValue = notice.CreatedByUser.Email
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}

	if reference.Schedule.ActivateFrom != notice.Schedule.ActivateFrom {
		df.FieldName = "Schedule.ActivateFrom"
		df.RefStrValue = reference.Schedule.ActivateFrom
		df.NoticeStrValue = notice.Schedule.ActivateFrom
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Schedule.ActivateTo != notice.Schedule.ActivateTo {
		df.FieldName = "Schedule.ActivateTo"
		df.RefStrValue = reference.Schedule.ActivateTo
		df.NoticeStrValue = notice.Schedule.ActivateTo
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Schedule.Hours != notice.Schedule.Hours {
		df.FieldName = "Schedule.Hours"
		df.RefStrValue = reference.Schedule.Hours
		df.NoticeStrValue = notice.Schedule.Hours
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Schedule.Days != notice.Schedule.Days {
		df.FieldName = "Schedule.Days"
		df.RefStrValue = reference.Schedule.Days
		df.NoticeStrValue = notice.Schedule.Days
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Schedule.WeekDays != notice.Schedule.WeekDays {
		df.FieldName = "Schedule.WeekDays"
		df.RefStrValue = reference.Schedule.WeekDays
		df.NoticeStrValue = notice.Schedule.WeekDays
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Schedule.Month != notice.Schedule.Month {
		df.FieldName = "Schedule.Month"
		df.RefStrValue = reference.Schedule.Month
		df.NoticeStrValue = notice.Schedule.Month
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}

	// bool
	if reference.Pdf != notice.Pdf {
		df.FieldName = "Pdf"
		df.RefBoolValue = reference.Pdf
		df.NoticeBoolValue = notice.Pdf
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Deleted != notice.Deleted {
		df.FieldName = "Deleted"
		df.RefBoolValue = reference.Deleted
		df.NoticeBoolValue = notice.Deleted
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.CreatedByUser.Active != notice.CreatedByUser.Active {
		df.FieldName = "CreatedByUser.Active"
		df.RefBoolValue = reference.CreatedByUser.Active
		df.NoticeBoolValue = notice.CreatedByUser.Active
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}
	if reference.Schedule.Deleted != notice.Schedule.Deleted {
		df.FieldName = "Schedule.Deleted"
		df.RefBoolValue = reference.Schedule.Deleted
		df.NoticeBoolValue = notice.Schedule.Deleted
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}

	// int64
	if reference.BackgroundFileId != notice.BackgroundFileId {
		df.FieldName = "BackgroundFileId"
		df.RefInt64Value = reference.BackgroundFileId
		df.NoticeInt64Value = notice.BackgroundFileId
		diff = append(diff, df)
		AddNoticesDiffToDB(df)
	}

	return diff, len(diff)
}

type TPlayListsDiff struct {
	gorm.Model
	PlayListsDiff
}

type PlayListsDiff struct {
	RPlayListId        int
	PlayListId         int
	FieldName          string
	RefIntValue        int
	PlayListIntValue   int
	RefStrValue        string
	PlayListStrValue   string
	RefBoolValue       bool
	PlayListBoolValue  bool
	RefUIntValue       uint
	PlayListUIntValue  uint
	RefInt64Value      int64
	PlayListInt64Value int64
}

func AddPlaylistDiffToDB(PlaylistDiff PlayListsDiff) {
	Df := TPlayListsDiff{
		PlayListsDiff: PlaylistDiff,
	}
	Db.Create(&Df)
}

func Compare2Playlists(reference, playlist TypePlaylist) (diff []PlayListsDiff, diffLength int) {
	var df PlayListsDiff
	df.RPlayListId = reference.Id
	df.PlayListId = playlist.Id

	// int
	if reference.Id != playlist.Id {
		df.FieldName = "Id"
		df.RefIntValue = reference.Id
		df.PlayListIntValue = playlist.Id
		diff = append(diff, df)
		AddPlaylistDiffToDB(df)
	}

	if reference.Announcements != playlist.Announcements {
		df.FieldName = "Announcements"
		df.RefIntValue = reference.Announcements
		df.PlayListIntValue = playlist.Announcements
		diff = append(diff, df)
		AddPlaylistDiffToDB(df)
	}

	// string
	if reference.Title != playlist.Title {
		df.FieldName = "Title"
		df.RefStrValue = reference.Title
		df.PlayListStrValue = playlist.Title
		diff = append(diff, df)
		AddPlaylistDiffToDB(df)
	}

	return diff, len(diff)

}
