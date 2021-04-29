// rld is a tool that watches a go program and automatically restart the
// application when a file change is detected.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

var (
	version = "0.0.1"
)
var usage = `Usage: rld <file>
Options:
  <file> - The filepath to watch for changes
`

func main() {
	if len(os.Args) < 2 {
		errUsage()
	}

	f := os.Args[1]

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	sigs := make(chan os.Signal, 1)
	echan := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// run command and watch file for changes
	info(f)
	go runCmd(f)

	go func() {
		for {
			select {
			case e := <-watcher.Events:
				if e.Op.String() == "WRITE" || e.Op.String() == "WRITE|CHMOD" {
					fmt.Println("[rld] detected change, restarting...")
					go runCmd(f)
				}
			case err := <-watcher.Errors:
				log.Fatal(err)
			case sig := <-sigs:
				fmt.Println()
				fmt.Println(sig)
				echan <- true
			}
		}
	}()

	if err := watcher.Add(f); err != nil {
		log.Fatal(err)
	}

	go func() { <-done }()
	<-echan
}

func info(f string) {
	fmt.Println("[rld] version=", version)
	fmt.Println("[rld] watching changes for", f)
}

func runCmd(file string) {
	fmt.Println("[rld] exec: go run", file)
	cmd := exec.Command("go", "run", file)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Print(cmd.Run())
}

func errUsage() {
	fmt.Fprintf(os.Stderr, usage)
	fmt.Fprintf(os.Stderr, "\n\n")
	os.Exit(1)
}
