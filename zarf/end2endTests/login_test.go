package end2endTests

import (
	"testing"
)

func TestLogin(t *testing.T) {
	finApp, err := new()
	if err != nil {
		t.Fatalf("%v", err)
	}

	err = finApp.StartBackground()
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Log("App started in background")

	err = finApp.Stop()
	if err != nil {
		t.Logf("error while stopping %v", err)
	}

}
