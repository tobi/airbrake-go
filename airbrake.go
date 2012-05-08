package airbrake

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"text/template"
)

var (
	ApiKey   = ""
	Endpoint = "https://airbrake.io/notifier_api/v2/notices.xml"
	Verbose  = false

	badResponse   = errors.New("Bad response")
	apiKeyMissing = errors.New("Please set the airbrake.ApiKey before doing calls")
	dunno         = []byte("???")
	centerDot     = []byte("·")
	dot           = []byte(".")
	tmpl          = template.Must(template.New("error").Parse(source))
)

type Line struct {
	Function string
	File     string
	Line     int
}

// stack implements Stack, skipping N frames
func stacktrace(skip int) (lines []Line) {
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		item := Line{string(function(pc)), string(file), line}
		lines = append(lines, item)
	}
	return
}

var channel chan map[string]interface{}
var once sync.Once

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//  runtime/debug.*T·ptrmethod
	// and want
	//  *T.ptrmethod
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

func initChannel() {
	channel = make(chan map[string]interface{}, 100)
	go worker()
}

func worker() {
	for params := range channel {
		buffer := bytes.NewBufferString("")

		if err := tmpl.Execute(buffer, params); err != nil {
			log.Printf("Airbreak error: %s", err)
			continue
		}

		response, err := http.Post(Endpoint, "text/xml", buffer)
		response.Body.Close()

		if err != nil {
			log.Printf("Airbreak error: %s", err)
			continue
		}

		if Verbose {
			log.Printf("Airbreak post: %s", params["Error"])
		}

	}

	return
}

func ErrorRequest(e error, request *http.Request) error {
	once.Do(initChannel)

	if ApiKey == "" {
		return apiKeyMissing
	}

	params := map[string]interface{}{
		"Error":     e,
		"ApiKey":    ApiKey,
		"ErrorName": e.Error(),
		"Request":   request,
	}

	pwd, err := os.Getwd()
	if err == nil {
		params["Pwd"] = pwd
	}

	params["Backtrace"] = stacktrace(3)

	channel <- params
	return nil
}

func Error(e error) error {
	return ErrorRequest(e, nil)
}

const source = `
<?xml version="1.0" encoding="UTF-8"?>
<notice version="2.0">
  <api-key>{{ .ApiKey }}</api-key>
  <notifier>
    <name>Airbrake Golang</name>
    <version>0.0.1</version>
    <url>http://airbrake.io</url>
  </notifier>
  <error>
    <class>{{ html .ErrorName }}</class>
    <message>test{{ with .ErrorMessage }}{{html .}}{{ end }}</message>
    <backtrace>
      {{ range .Backtrace }}
      <line method="{{.Function}}" file="{{.File}}" number="{{.Line}}"/>
      {{ end }}
    </backtrace>
  </error>
  {{ with .Request }}
  <request>
    <url>{{ .URL }}</url>
    <component/>
    <action/>
    <cgi-data>
    </cgi-data>
  </request>
  {{ end }}  
  <server-environment>
    <environment-name>production</environment-name>
    <project-root>{{ .Pwd }}</project-root>        
  </server-environment>
</notice>`
