package main

import (
	movetoroutines "billionRowChallenge/moveToRoutines"
	multireader "billionRowChallenge/multiReader"
	"billionRowChallenge/parsers"
	"billionRowChallenge/utilities"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// main - Core entry point to the Billion Row Challenge
func main() {

	// start := time.Now()

	// noroutines.NoRoutineMain()
	movetoroutines.BuildRoutinesMain()

	return

	// Get the current working directory
	// TODO: Strip this out before the competition and hard-code the path to the file to speed up execution.
	goExecutable, err := os.Executable()
	if err != nil {
		panic(err)
	}
	executablePath := filepath.Dir(goExecutable)

	// Check the current file size in number of bytes
	// TODO: See if this step is faster than monitoring for an EOF during the parsing
	f, err := os.Stat(filepath.Join(executablePath, "measurements.csv"))
	if err != nil {
		panic(err)
	}
	bytesInFile := f.Size()
	numberOfRoutineCalls := bytesInFile / utilities.BufferSize // Will return an int64 value

	finalBufferSize := bytesInFile - (numberOfRoutineCalls * utilities.BufferSize)

	// file, err := os.Open(filepath.Join(executablePath, "measurements.csv"))
	file, err := os.Open(filepath.Join(executablePath, "measurements.csv"))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			panic(err)
		}
	}()

	// Launch a number of routines that will sit and process read buffers as they arrive
	for _ = range utilities.NumberOfReaderRoutines {
		go multireader.PartialReader(file, utilities.BufferSize)
	}

	go parsers.CombinePartialReads()

	// Start up the channel calls
	for i := range int64(numberOfRoutineCalls) {
		multireader.ReaderChan <- multireader.ReaderFields{
			Offset: i * utilities.BufferSize,
			Index:  i,
		}

		if i == numberOfRoutineCalls {
			fmt.Println("Basically done")
			// fmt.Println(time.Since(start))
		}
	}

	if finalBufferSize > 0 {
		// go multireader.FinalReader(file, finalBufferSize, numberOfRoutineCalls*utilities.BufferSize, int64(numberOfRoutineCalls))
		go multireader.FinalReader(file, finalBufferSize, numberOfRoutineCalls*utilities.BufferSize, int64(numberOfRoutineCalls))
	}

	time.Sleep(time.Minute * 5)
}
