package include

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

var client *http.Client

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AuthHeader := r.Header.Get("Auth")
		if AuthHeader == os.Getenv("AUTH_HASH") {
			next.ServeHTTP(w, r)
		}
	})
}

func PostTelegrammMessage(msg string) {

	var url string

	teleBotID := os.Getenv("TELE_BOT_ID")
	teleBotChannel := os.Getenv("TELE_BOT_CHANNEL")

	// fmt.Println(msg)

	if teleBotID != "" && teleBotChannel != "" {

		url = "https://api.telegram.org/bot" + teleBotID + "/sendMessage?chat_id=" + teleBotChannel + "&parse_mode=HTML&text="

		msg = strings.Replace(msg, " ", "+", -1)
		msg = strings.Replace(msg, "'", "%27", -1)
		msg = strings.Replace(msg, "\n", "%0A", -1)

		url = url + msg
		fmt.Println("\n" + url + "\n")
		response, _ := http.Get(url)
		fmt.Println(response)

	}
}

func HeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		//w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("content-type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func ResponseOK(w http.ResponseWriter, addedRecordString []byte) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	n, _ := fmt.Fprintf(w, string(addedRecordString))
	fmt.Println("Response was sent ", n, " bytes")
	return
}

func ResponseBadRequest(w http.ResponseWriter, err error, message string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("content-type", "application/json")
	errorString := "{\"error_message\":\"" + err.Error() + "\",\"message\":\"" + message + "\"}"
	http.Error(w, errorString, http.StatusBadRequest)
	return
}

func ResponseNotFound(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	n, _ := fmt.Fprintf(w, "")
	fmt.Println("Response was sent ", n, " bytes")
	return
}

func ResponseUnknown(w http.ResponseWriter, message string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("content-type", "application/json")
	errorString := "{\"message\":\"" + message + "\"}"
	http.Error(w, errorString, http.StatusInternalServerError)
	return
}
