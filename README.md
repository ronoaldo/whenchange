# whenchange

[![GoDoc](https://godoc.org/ronoaldo.gopkg.net/whenchange?status.png)](https://godoc.org/ronoaldo.gopkg.net/whenchange)

Command whenchange monitors for changes on files, directories, and optionally
watching sub-directories, and when a change happens, executes a command.

### Installation

    go get ronoaldo.gopkg.net/whenchange

### Usage

    whenchange -p source.go go build

The above command will monitor all go files in the current directory for
changes, and trigger go build.

    whenchange -p ./src/ mvn test-compile

The above command will monitor recursivelly the src folder, and execute the
maven test compile target.
