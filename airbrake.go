// Package airbrake sends errors and panics to http://airbrake.io and compatible services.
package airbrake

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"text/template"
)

var (
	ApiKey      = ""
	Hostname    = ""
	ProjectRoot = ""
	Environment = "development"
	Version     = ""
	Endpoint    = "https://api.airbrake.io/notifier_api/v2/notices" // not all plans support HTTPS!
	Verbose     = false                                             // uses log.Print if set to true

	badResponse   = errors.New("Bad response")
	apiKeyMissing = errors.New("Please set the airbrake.ApiKey before doing calls")
	tmpl          = template.Must(template.New("error").Parse(source))
)

func init() {
	hostname, err := os.Hostname()
	if err == nil {
		Hostname = hostname
	}

	pwd, err := os.Getwd()
	if err == nil {
		ProjectRoot = pwd
	}
}

type Line struct {
	Function string
	File     string
	Line     int
}

// return backtrace, skipping some lines
func backtrace(skip int) (lines []Line) {
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		item := Line{function(pc), string(file), line}

		// ignore panic method
		if item.Function != "panic" {
			lines = append(lines, item)
		}
	}
	return
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) (name string) {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "???"
	}

	name = fn.Name()

	// Remove import path from name: reduce
	// "github.com/tobi/airbrake-go_test.(*S).f" to "airbrake-go_test.(*S).f"
	if period := strings.LastIndex(name, "/"); period >= 0 {
		name = name[period+1:]
	}

	// center dot is used in internals instead of dot
	name = strings.Replace(name, "·", ".", -1)
	return
}

func post(params map[string]interface{}) {
	buffer := new(bytes.Buffer)

	if err := tmpl.Execute(buffer, params); err != nil {
		log.Printf("Airbreak error: %s", err)
		return
	}

	if Verbose {
		log.Printf("Airbreak payload for endpoint %s: %s", Endpoint, buffer)
	}

	response, err := http.Post(Endpoint, "text/xml", buffer)
	if err != nil {
		log.Printf("Airbreak error: %s", err)
		return
	}
	defer response.Body.Close()

	if Verbose || response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		log.Printf("Airbreak response: %s", body)
		log.Printf("Airbreak post: %q (%s) status code: %d", params["Error"], params["Class"], response.StatusCode)
	}
}

func makeParams(e error) (params map[string]interface{}) {
	params = map[string]interface{}{
		"Class":     reflect.TypeOf(e).String(),
		"Error":     e,
		"ApiKey":    ApiKey,
		"ErrorName": e.Error(),
	}

	if params["Class"] == "" {
		params["Class"] = "Panic"
	}

	params["Hostname"] = Hostname
	params["ProjectRoot"] = ProjectRoot
	params["Environment"] = Environment
	params["Version"] = Version

	return
}

// Send error with request information and backtrace.
func Error(e error, request *http.Request) error {
	if ApiKey == "" {
		return apiKeyMissing
	}

	params := makeParams(e)
	params["Request"] = request
	params["Backtrace"] = backtrace(2)

	post(params)
	return nil
}

// Notify about error (without backtrace).
func Notify(e error) error {
	if ApiKey == "" {
		return apiKeyMissing
	}

	params := makeParams(e)

	post(params)
	return nil
}

func CapturePanic(r *http.Request) {
	if rec := recover(); rec != nil {
		err, ok := rec.(error)
		if !ok {
			err = fmt.Errorf("%v", rec)
		}

		log.Printf("Recording error %s %T", err, rec)
		Error(err, r)

		panic(rec)
	}
}

// current schema: http://airbrake.io/airbrake_2_4.xsd
const source = `<?xml version="1.0" encoding="UTF-8"?>
<notice version="2.0">
  <api-key>{{ .ApiKey }}</api-key>
  <notifier>
    <name>Airbrake Golang</name>
    <version>0.0.1</version>
    <url>https://github.com/tobi/airbrake-go</url>
  </notifier>

  <error>
    <class>{{ html .Class }}</class>
    <message>{{ html .ErrorName }}</message>
    <backtrace>{{ range .Backtrace }}
      <line method="{{ html .Function }}" file="{{ html .File }}" number="{{ .Line }}"/>{{ end }}
    </backtrace>
  </error>
  {{ with .Request }}
  <request>
    <url>{{ html .URL }}</url>
    <component/>
    <action/>
  </request>
  {{ end }}
  <server-environment>
    <project-root>{{ html .ProjectRoot }}</project-root>
    <environment-name>{{ html .Environment }}</environment-name>
    <app-version>{{ html .Version }}</app-version>
    <hostname>{{ html .Hostname }}</hostname>
  </server-environment>
</notice>`
