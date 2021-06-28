// rld is a tool that watches a go program and automatically restart the
// application when a file change is detected.
package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	version = "0.0.1"
	process *os.Process
)

var usage = `Usage: rld <file>
Options:
  <file> - The filepath to watch for changes
`

func main() {
	// if len(os.Args) < 2 {
	// 	errUsage()
	// }

	var (
		check fs.FileInfo
		err   error
		path  string
		args  []string
	)
	if len(os.Args) < 2 {
		path = "."
	} else {
		path = os.Args[1]
	}

	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	check, err = os.Stat(path)
	if err != nil {
		log.Fatal("1", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("2", err)
	}
	defer watcher.Close()

	waiting := false
	timer := time.NewTimer(1000 * time.Millisecond)
	sigs := make(chan os.Signal, 1)
	echan := make(chan struct{})
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	filesToSkip := []string{"vendor"}

	// run command and watch file for changes
	//info(check.Name())

	go func() {
		for {
			select {
			case e := <-watcher.Events:
				if e.Op.String() == "WRITE" || e.Op.String() == "WRITE|CHMOD" {
					if waiting {
						if !timer.Stop() {
							<-timer.C
						}
						timer.Reset(500 * time.Millisecond)
						continue
					}
					fmt.Println("[rld] detected change")
					fmt.Println("[rld] waiting for 500ms to verify file closure")
					//sets waiting to true for subsequent write events
					waiting = true
					timer.Reset(500 * time.Millisecond)
				}

			case <-timer.C:
				if waiting {
					fmt.Println("[rld] no further change detected, restarting...")
					killPid(process)
					go runCmd(path)
					waiting = false
				}

			case err := <-watcher.Errors:
				log.Fatal("3", err)

			case sig := <-sigs:
				fmt.Println()
				fmt.Println(sig)
				close(echan)
			}
		}
	}()

	go func() {
		var input string
		for {
			fmt.Scanln(&input)
			if input == "rst" {
				fmt.Println("[rld] manual input requested, restarting...")
				killPid(process)
				go runCmd(path)
			}
		}
	}()

	if check.IsDir() {
		fmt.Println("[rld] Directory detected")
		fmt.Println(path)

		if path != "." {
			err := os.Chdir(path)
			if err != nil {
				log.Fatal("4", err)
			}
			path = "."
		}

		if _, err := os.Stat(path + "/go.mod"); err != nil {
			log.Fatal("No go.mod File Found In Directory, Exiting")
		}
		filepath.Walk(path, func(docpath string, fileinfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fileinfo.IsDir() {
				if contains(filesToSkip, fileinfo.Name()) || fileinfo.Name() != "." && strings.HasPrefix(fileinfo.Name(), ".") {
					fmt.Printf("Skipping Dir %v\n", fileinfo.Name())
					return filepath.SkipDir
				}

			} else {
				if filepath.Ext(fileinfo.Name()) == ".go" && !strings.Contains(fileinfo.Name(), "_test") {
					info(docpath)
					err = watcher.Add(path)
				}
			}

			return err
		})
		go runCmd(path)
	} else {
		fmt.Println("[rld] File detected")
		info(path)
		if err := watcher.Add(check.Name()); err != nil {
			log.Fatal("5", err)
		}
		go runCmd(fmt.Sprintf("%s %s", path, strings.Join(args, " ")))
	}

	//	go func() { <-done }()
	<-echan

	//Kill Last Created Child Before Exiting
	killPid(process)
}

func info(f string) {
	fmt.Println("[rld] watching changes for", f)
}

func runCmd(file string) {
	fmt.Println("[rld] exec: go run", file)
	args := []string{"run"}
	args = append(args, strings.Fields(file)...)
	cmd := exec.Command("go", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Print(cmd.Start(), "\nProgram Output:\n==============\n")

	if cmd.Process != nil {
		process = cmd.Process
	}
}

func errUsage() {
	fmt.Fprintf(os.Stderr, usage)
	fmt.Fprintf(os.Stderr, "\n\n")
	os.Exit(1)
}

func killPid(process *os.Process) {
	fmt.Printf("[rld] Killing previous process: %d\n", process.Pid)
	if process != nil {
		syscall.Kill(-process.Pid, syscall.SIGKILL)
	}
}

func contains(arr []string, elem string) bool {
	if strings.HasPrefix(elem, ".") {
		return false
	}

	for _, i := range arr {
		if elem == i {
			return true
		}
	}
	return false
}
