package end2endTests

import "testing"

func TestLogin(t *testing.T) {
	finApp, err := new()
	if err != nil {
		t.Fatalf("%v", err)
	}
	_ = finApp
	t.Log(finApp)

}
