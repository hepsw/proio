package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/decibelcooper/proio/go-proio"
)

var (
	doGzip = flag.Bool("g", false, "decompress the stdin input with gzip")
	event  = flag.Int("e", -1, "list specified event, numbered consecutively from the start of the file or stream")
)

func printUsage() {
	fmt.Fprintf(os.Stderr,
		`Usage: proio-ls [options] <proio-input-file>
options:
`,
	)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = printUsage
	flag.Parse()

	if flag.NArg() != 1 {
		printUsage()
		log.Fatal("Invalid arguments")
	}

	var reader *proio.Reader
	var err error

	filename := flag.Arg(0)
	if filename == "-" {
		stdin := bufio.NewReader(os.Stdin)
		if *doGzip {
			reader, err = proio.NewGzipReader(stdin)
		} else {
			reader = proio.NewReader(stdin)
		}
	} else {
		reader, err = proio.Open(filename)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	singleEvent := false
	if *event >= 0 {
		singleEvent = true
		_, err = reader.Skip(*event)
		if err == proio.ErrResync {
			log.Print(err)
		} else if err != nil {
			log.Fatal(err)
		}
	}

	nEventsRead := 0

	for event := range reader.ScanEvents() {
		fmt.Print(event)

		nEventsRead++
		if singleEvent {
			reader.StopScan()
			break
		}
	}

errLoop:
	for {
		select {
		case err := <-reader.Err:
			if err != io.EOF || nEventsRead == 0 {
				log.Print(err)
			}
		default:
			break errLoop
		}
	}
}
