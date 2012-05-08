package airbreak

import (
  "errors"
  "testing"
//  "testing/quick"
  "net/http"
  "bytes"
)

func TestError(t *testing.T) {
  
  err := Error(errors.New("GenericFailure"))
  if err != nil {
    
    t.Error(err)
  }
}

func TestRequest(t *testing.T) {

  request, _ := http.NewRequest("GET", "/some/path?a=1", bytes.NewBufferString(""))
  
  err := ErrorRequest(errors.New("GenericFailure"), request)
  
  if err != nil {    
    t.Error(err)
  }
}
