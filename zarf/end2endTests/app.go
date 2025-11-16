package end2endTests

import (
	"fmt"
	"os"
	"path/filepath"
)

type app struct {
	shutdown chan struct{}
	path     string
}

func new() (*app, error) {

	path, err := rootPath()
	if err != nil {
		return nil, err
	}

	a := app{
		shutdown: make(chan struct{}),
		path:     path,
	}
	return &a, nil
}

func (app app) StartBackground() {

}

// rootPath returns the root of the project
func rootPath() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to get current directory: %w", err)
	}
	pwd = filepath.Join(filepath.Dir(pwd), "../..")
	abs, err := filepath.Abs(pwd)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute path: %w", err)
	}

	return abs, nil
}

func startBackend() {
	//app := "echo"
	//
	//arg0 := "-e"
	//arg1 := "Hello world"
	//arg2 := "\n\tfrom"
	//arg3 := "golang"
	//
	//cmd := exec.Command(app, arg0, arg1, arg2, arg3)
	//stdout, err := cmd.Output()
	//
	//if err != nil {
	//	fmt.Println(err.Error())
	//	return
	//}
}

func startFrontend() {

}

func (app app) Stop() {

}
