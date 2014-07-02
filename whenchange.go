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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	// List of paths to watch
	pathList PathList
	// Watch directory recursively
	recursive bool
	// If we should print the output of executed command
	printOut bool
	// Command to execute on changes
	cmd []string
	// Debug options
	debug bool
	// fsnotify.Watcher to monitor changes
	watcher *fsnotify.Watcher
	// Shell to use when running the command
	shell string
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
	flag.BoolVar(&debug, "d", false, "Output debug information")
	flag.Var(&pathList, "p", "Files and directories to watch")
	flag.BoolVar(&printOut, "out", true, "Print output of executed command")
	flag.StringVar(&shell, "shell", "bash", "The shell to use when running the command")
	flag.Usage = func() {
		w := os.Stderr
		fmt.Fprintf(w, "whenchange - run shell command when files change\n")
		fmt.Fprintf(w, "Usage: whenchange [-d] [-r|--recursive] "+
			"[-out] [-p path [-p path]...] command\n")
		fmt.Fprintf(w, "All positional arguments will compose the resulting command to execute")
		flag.PrintDefaults()
	}
}

func main() {
	// Parse and print help
	flag.Parse()
	cmd = flag.Args()
	Debugf("Command to execute: %v", cmd)

	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	if len(pathList) < 1 {
		pathList.Set("./")
	}

	Debugf("Path list %v", pathList)

	for _, f := range pathList {
		Watch(f)
		for _, s := range SubDirs(f) {
			Watch(s)
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
	Debugf("%s changed (%s)", path, ev)

	if ev.IsCreate() {
		if IsDir(path) {
			Debugf("Monitoring %s", path)
			watcher.Watch(path)
		}
	}

	// Run command
	if len(cmd) > 0 {
		out, err := exec.Command(shell, "-c", strings.Join(cmd, " ")).CombinedOutput()
		if err != nil {
			log.Printf("Error running command %s: %s", cmd, err)
		}
		if printOut && len(out) > 0 {
			log.Printf("Command output:\n%s\n", string(out))
		}
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

// Watch watches and logs if any error happens.
func Watch(f string) {
	Debugf("Watching [%s]", f)
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

func Debugf(f string, args ...interface{}) {
	if debug {
		log.Printf(f, args...)
	}
}
