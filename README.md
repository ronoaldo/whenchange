# whenchange
--
Command whenchange monitors for changes on files, directories, and optionally
watching sub-directories, and when a change happens, executes a command.

### Installation

    go get ronoaldo.goimport.net/whenchange

### Usage

    whenchange *.go go build

The above command will monitor all go files in the current directory for
changes, and trigger go build.

    whenchange ./src/ mvn test-compile

The above command will monitor recursivelly the src folder, and execute the
maven test compile target.
