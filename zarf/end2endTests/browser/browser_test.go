package browser

import (
	"github.com/davecgh/go-spew/spew"
	"testing"
)

// prevent IDE to delete module
var _ = spew.Dump

func TestStart(t *testing.T) {
	br, err := New(Cfg{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("unexpected start error %v", err)
	}

	t.Logf("using Browser: %s", br.Path)

	err = br.Start()
	if err != nil {
		t.Fatalf("unexpected start error %v", err)
	}
}

func TestNavigate(t *testing.T) {
	br, _ := New(Cfg{
		Headless: true,
	})
	_ = br.Start()

	_, err := br.Navigate("https://example.com")
	if err != nil {
		t.Fatalf("unable to navigate to a page: %v", err)
	}
}
