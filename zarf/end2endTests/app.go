package end2endTests

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type app struct {
	shutdown    chan struct{}
	projectPath string

	backendCmd *exec.Cmd
}

func new() (*app, error) {

	path, err := rootPath()
	if err != nil {
		return nil, err
	}

	a := app{
		shutdown:    make(chan struct{}),
		projectPath: path,
	}
	return &a, nil
}

// rootPath returns the root of the project
func rootPath() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to get current directory: %w", err)
	}

	file := "go.mod"
	// check if we are in the root of the dir
	abs, err := filepath.Abs(pwd)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute path: %w", err)
	}
	abs = filepath.Clean(abs)
	if fileExists(filepath.Join(pwd, file)) {
		return abs, nil
	}

	// check if we are in zarf dir
	abs = filepath.Join(abs, "..")
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute path: %w", err)
	}
	abs = filepath.Clean(abs)
	if fileExists(filepath.Join(abs, file)) {
		return abs, nil
	}

	// check if we are in end2endTests dir
	abs = filepath.Join(abs, "..")
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute path: %w", err)
	}
	abs = filepath.Clean(abs)
	if fileExists(filepath.Join(abs, file)) {
		return abs, nil
	}
	return "", fmt.Errorf("unable to find project root directory, last checked %s", abs)
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else if errors.Is(err, os.ErrNotExist) {
		return false
	} else {
		panic(err)
	}
}

type saveOutput struct {
	savedOutput []byte
}

func (so *saveOutput) Write(p []byte) (n int, err error) {
	so.savedOutput = append(so.savedOutput, p...)
	return os.Stdout.Write(p)
}

func (app *app) StartBackground() error {
	err := app.startBackend()
	if err != nil {
		return err
	}

	// wait for the processes to start
	time.Sleep(1 * time.Second)
	return nil

}

func (app *app) startBackend() error {

	var so saveOutput

	args := []string{"make", "run"}
	cmd := exec.Command(args[0], args[1:]...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatalf("could not get stderr pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("could not get stdout pipe: %v", err)
	}

	//cmd.Stdin = os.Stdin
	//cmd.Stdout = &so
	//cmd.Stderr = os.Stderr

	cmd.Dir = app.projectPath

	err = cmd.Start()
	if err != nil {
		return err
	}
	go func() {
		werr := cmd.Wait()
		if werr != nil {
			_ = cmd.Process.Kill()
			spew.Dump(werr)

			panic(werr)
		}
	}()
	app.backendCmd = cmd

	return nil
}

func startFrontend() {

}

func (app *app) Stop() error {
	err := app.backendCmd.Process.Kill()
	if err != nil {
		return err
	}
	return nil
}
