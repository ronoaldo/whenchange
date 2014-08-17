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
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/fsnotify.v0"
)

var (
	// List of paths to watch
	pathList PathList
	// Watch directory recursively
	recursive bool
	// Command to execute on changes
	cmd []string
	// verbose options
	verbose bool
	// fsnotify.Watcher to monitor changes
	watcher *Watcher
	// Shell to use when running the command
	shell string
	// Delay between repeated executions of command
	delaySpec string
	delay     time.Duration
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

type Watcher struct {
	*fsnotify.Watcher
	list   map[string]time.Time
	listMu sync.Mutex
}

func (w *Watcher) Watch(path string) {
	w.listMu.Lock()
	defer w.listMu.Unlock()
	if _, ok := w.list[path]; ok {
		verbosef("Path %s already in watch list", path)
		return
	}
	verbosef("Watching [%s]", path)
	err := w.Watcher.Watch(path)
	if err != nil {
		log.Fatal(err)
	}
	// To prevent ignoring the very first change, use a time machine and
	// go back in time :D
	w.list[path] = time.Now().Add(-5 * time.Second)
}

func init() {
	flag.StringVar(&delaySpec, "delay", "5s", "Delay between repeated executions of command")
	flag.StringVar(&delaySpec, "d", "5s", "Delay between repeated executions of command (shorthand)")
	flag.BoolVar(&recursive, "recursive", true, "Watch directories recursively")
	flag.BoolVar(&recursive, "r", true, "Watch directories recursively (shorthand)")
	flag.BoolVar(&verbose, "verbose", false, "Output verbose information")
	flag.BoolVar(&verbose, "v", false, "Output verbose information (shorthand)")
	flag.Var(&pathList, "path", "Files and directories to watch")
	flag.Var(&pathList, "p", "Files and directories to watch (shorthand)")
	flag.StringVar(&shell, "shell", "bash", "The shell to use when running the command")
	flag.Usage = func() {
		w := os.Stderr
		fmt.Fprintf(w, "Usage: whenchange [options] commands\n")
		fmt.Fprintf(w, "All positional arguments will compose the resulting command to execute\n")
		fmt.Fprintf(w, "Options can be:\n")
		flag.PrintDefaults()
	}
}

func main() {
	// Parse and print help
	flag.Parse()
	cmd = flag.Args()
	verbosef("Command to execute: %v", cmd)

	var err error
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	watcher = &Watcher{Watcher: fsw, list: make(map[string]time.Time)}
	defer watcher.Close()

	if len(pathList) < 1 {
		pathList.Set("./")
	}

	delay, err = time.ParseDuration(delaySpec)
	if err != nil {
		log.Printf("Invalid duration: %s. Using 5s instead", delaySpec)
		delay = 5 * time.Second
	}

	verbosef("Path list %v", pathList)

	for _, f := range pathList {
		watcher.Watch(f)
		for _, s := range SubDirs(f) {
			watcher.Watch(s)
		}
	}

	for {
		select {
		case ev := <-watcher.Event:
			HandleEvent(ev)
		case err := <-watcher.Error:
			HandleError(err)
		}
	}
}

// Func HandleEvent monitors for changes, executes the specified command
// and keep monitoring for new folders when added.
func HandleEvent(ev *fsnotify.FileEvent) {
	path := filepath.Clean(ev.Name)
	verbosef("%s changed (%s)", path, ev)
	if ev.IsCreate() {
		if IsDir(path) {
			verbosef("Monitoring %s", path)
			watcher.Watch(path)
		}
	}
	if ev.IsDelete() {
		// TODO: Handle directory removal -- do we need to ignore children?
		verbosef("Stopping watching for %s", path)
		watcher.RemoveWatch(path)
	}
	// Locking, because we will change the path map
	watcher.listMu.Lock()
	defer watcher.listMu.Unlock()
	now := time.Now()
	if now.Sub(watcher.list[path]) < delay {
		verbosef("File %s changed too fast. Ignoring this change.", path)
		return
	}
	watcher.list[path] = now
	// Run command
	if len(cmd) > 0 {
		c := strings.Join(cmd, " ")
		log.Printf("Running command '%s' ...", c)
		cmd := exec.Command(shell, "-c", c)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Printf("Error: %s", err)
		}
		log.Printf("Done.")
	} else {
		log.Printf("No command to run.")
	}
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		log.Printf("Unable to stat %s: %v", path, err)
		return false
	}
	return s.IsDir()
}

// Handle any errors when they happend.
func HandleError(err error) {
	log.Printf(err.Error())
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

func verbosef(f string, args ...interface{}) {
	if verbose {
		log.Printf(f, args...)
	}
}
