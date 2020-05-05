package include

import (
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
)

var homeTempl = template.Must(template.New("").Parse(homeHTML))

func GetTestResult(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	switch r.Method {
	case "GET":

		var test Test
		Db.Where("hash = ?", params["id"]).Last(&test)

		var testError []TestError
		Db.Where("test_id = ?", test.ID).Find(&testError)

		w.Header().Set("content-type", "text/html")
		homeTempl.Execute(w, &testError)

	}
}

const homeHTML = `<!DOCTYPE html>
<html lang="en">
    <head>
          <title>Condomanager Ftp</title>
		  <meta charset="utf-8">
		  <meta name="viewport" content="width=device-width, initial-scale=1">
		  <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.4.1/css/bootstrap.min.css">
		  <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.4.1/jquery.min.js"></script>
		  <script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.16.0/umd/popper.min.js"></script>
		  <script src="https://maxcdn.bootstrapcdn.com/bootstrap/4.4.1/js/bootstrap.min.js"></script>
    </head>
    <body>

		<div>
  		<h2>List Of Errors</h2>
        <table class="table table-striped table-striped table-hover">
			<!-- Table body -->
			<thead>
			  <tr>
				<th>Test ID</th>
				<th>Type</th>
				<th>Message</th>
				<th>Description</th>
			  </tr>
			</thead>

			<tbody>
			{{ range . }}
			<tr>
				<td>{{ .TestId }}</td>
				<td>{{ .Type }}</td>
				<td>{{ .Message }}</td>
				<td>{{ .Description }}</td>
			</tr>
			{{ end }}
			</tbody>
		</table>
		</div>
    </body>
</html>

<style>
th {
	position: sticky;
	top: 0;
	z-index: 1;
	background-color: white;
}
</style>

`
