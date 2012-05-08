package airbreak

import (
  "bytes"
  "runtime"
  "errors"
  "log"
  "os"
  "text/template"
  "net/http"
)

var ApiKey string =  ""  
var Endpoint string = "https://airbreak.io/notifier_api/v2/notices.xml"

const source = `<?xml version="1.0" encoding="UTF-8"?>
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
      <var key="SERVER_NAME">example.org</var>
    </cgi-data>
  </request>
  {{ end }}  
  <server-environment>
    <environment-name>production</environment-name>
    <project-root>{{ .Pwd }}</project-root>        
  </server-environment>
</notice>`

var tmpl = template.Must(template.New("error").Parse(source))

var (
  badResponse = errors.New("Bad response")
  apiKeyMissing = errors.New("Please set the airbreak.ApiKey before doing calls")
  dunno     = []byte("???")
  centerDot = []byte("·")
  dot       = []byte(".")
)

type ExtendedError interface {
  Message() string
}

type Line struct { 
  Function string
  File string
  Line int
}

// stack implements Stack, skipping N frames
func stacktrace(skip int) (lines []Line) {
  for i := skip; ; i++ { 
    pc, file, line, ok := runtime.Caller(i)
    if !ok {
      break
    }

    item := Line{ string(function(pc)), string(file), line}
    lines = append(lines, item )
  }

  return
}

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

func ErrorRequest(e error, request *http.Request) error {
  if ApiKey == "" {
    return apiKeyMissing
  }

  params := map[string]interface{} { 
    "Error": e, 
    "ApiKey": ApiKey, 
    "ErrorName": e.Error(),
    "Request": request,
  }

  pwd, err := os.Getwd()
  if err == nil {
    params["Pwd"] = pwd
  }

  params["Backtrace"] = stacktrace(3)

  // if err := e.(ExtendedError); err == nil {
  //   params["ExtendedError"] = ExtendedError(err).Message()
  // } else {
  //   params["ExtendedError"] = e.Error()
  // }

  buffer := bytes.NewBufferString("")

  if err := tmpl.Execute(buffer, params); err != nil {    
    return err
  }

  log.Printf("%s", buffer)

  response, err := http.Post(Endpoint, "text/xml", buffer)
  defer response.Body.Close()

  if response.StatusCode != 201 && response.StatusCode != 200  {
    return badResponse
  }

  return nil
}

func Error(e error) error {
  return ErrorRequest(e, nil)
}
