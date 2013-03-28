package airbrake_test

import (
	. "."
	"bytes"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"
)

func init() {
	ApiKey = os.Getenv("AIRBRAKE_TEST_KEY")
	if ApiKey == "" {
		panic("Set environment variable AIRBRAKE_TEST_KEY")
	}

	endpoint := os.Getenv("AIRBRAKE_TEST_ENDPOINT")
	if endpoint != "" {
		Endpoint = endpoint
	}

	Environment = "testing"
	Verbose = true
}

func TestError(t *testing.T) {
	err := Error(errors.New("Test Error"), nil)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second) // to prevent throttling
}

func TestRequest(t *testing.T) {
	request, _ := http.NewRequest("GET", "/some/path?a=1", bytes.NewBufferString(""))

	err := Error(errors.New("Test Error"), request)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second)
}

func TestNotify(t *testing.T) {
	err := Notify(errors.New("Test Notify"))
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second)
}
