// Command whenchange monitors for changes on files, directories,
// and optionally watching sub-directories, and when a change
// happens, executes a command.
//
//
// Installation
//
//     go get ronoaldo.gopkg.net/whenchange
// 
// 
// Usage
//
//     whenchange -p source.go go build
//
// The above command will monitor all go files in the current
// directory for changes, and trigger go build.
//
//     whenchange -p ./src/ mvn test-compile
//
// The above command will monitor recursivelly the src folder,
// and execute the maven test compile target.
package main

import (
	"code.google.com/p/go.exp/fsnotify"
	"flag"
	"fmt"
	"path/filepath"
	"log"
	"os"
	"os/exec"
)

var (
	// List of paths to watch
	pathList PathList
	// Watch directory recursively
	recursive bool
	// Command to execute on changes
	cmd []string
	// fsnotify.Watcher to monitor changes
	watcher *fsnotify.Watcher
)

// Type PathList represents a set of paths to watch for.
type PathList []string

// Method String implements the flags.Value interface.
func (p *PathList) String() string {
	return fmt.Sprint(*p)
}

// Method Set implements the flags.Value interface.
func (p *PathList) Set(value string) error {
	*p = append(*p, value)
	return nil
}

func init() {
	flag.BoolVar(&recursive, "recursive", true, "Watch directories recursively")
	flag.BoolVar(&recursive, "r", true, "Watch directories recursively (shorthand)")
	flag.Var(&pathList, "p", "Files and directories to watch")
}

func main() {
	// Parse and print help
	flag.Parse()
	cmd = flag.Args()
	
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	
	if len(pathList) < 1 {
		pathList.Set("./")
	}
	
	for _, f := range(pathList) {
		for _, s := range(SubDirs(f)) {
			Watch(s)
		}
	}
	
	for {
		select {
			case ev := <- watcher.Event:
			HandleEvent(ev)
			case err := <- watcher.Error:
			HandleError(err)
		}
	}
}

// Func HandleEvent monitors for changes, executes the specified command
// and keep monitoring for new folders when added.
func HandleEvent(ev *fsnotify.FileEvent) {
	path := filepath.Clean(ev.Name)
	log.Printf("%s changed\n", path)
	
	if ev.IsCreate() {
		s, err := os.Stat(path)
		if err != nil {
			log.Printf("Error: %s: %s", path, err.Error())
			return
		}
		if s.IsDir() {
			log.Printf("Monitoring %s", path)
			watcher.Watch(path)
		}
	}
	
	// Run command
	if len(cmd) > 0 {
		out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			log.Printf("Error running command [%s]:\n%s", cmd, err)
			return
		}
		log.Printf(string(out))
	}
}

// Handle any errors when they happend.
func HandleError(err error) {
	log.Printf(err.Error())
}

// Watch watches and logs if any error happens.
func Watch(f string) {
	log.Println("Watching [%s]", f)
	err := watcher.Watch(f)
	if err != nil {
		log.Fatal(err)
	}
}

// Given a file path, all sub directories are returned.
func SubDirs(path string) []string {
	var paths []string
	filepath.Walk(path, func(newPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() {
			paths = append(paths, newPath)
		}
		
		return nil
	})
	return paths
}
