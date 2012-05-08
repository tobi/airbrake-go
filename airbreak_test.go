package airbreak

import (
  "errors"
  "testing"
//  "testing/quick"
  "net/http"
  "bytes"
  "time"
)

func TestError(t *testing.T) {
  Verbose = true
  ApiKey = ""
  Endpoint = "https://exceptions.shopify.com/notifier_api/v2/notices.xml"
  
  err := Error(errors.New("GenericFailure"))
  if err != nil {
    
    t.Error(err)
  }

  time.Sleep(1e9)
}

func TestRequest(t *testing.T) {
  Verbose = true
  ApiKey = ""
  Endpoint = "https://exceptions.shopify.com/notifier_api/v2/notices.xml"

  request, _ := http.NewRequest("GET", "/some/path?a=1", bytes.NewBufferString(""))
  
  err := ErrorRequest(errors.New("GenericFailure"), request)
  
  if err != nil {    
    t.Error(err)
  }

  time.Sleep(1e9)
}
