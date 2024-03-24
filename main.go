package main

import (
	multireader "billionRowChallenge/multiReader"
	"billionRowChallenge/output"
	"billionRowChallenge/parsers"
	"billionRowChallenge/utilities"
	"fmt"
	"os"
	"sync"
	"time"
)

// main - Core entry point to the Billion Row Challenge
func main() {

	// start := time.Now()

	var waitGroup sync.WaitGroup

	// noroutines.NoRoutineMain()
	// movetoroutines.BuildRoutinesMain()

	// Get the current working directory
	// TODO: Strip this out before the competition and hard-code the path to the file to speed up execution.
	// goExecutable, err := os.Executable()
	// if err != nil {
	// 	panic(err)
	// }
	// executablePath := filepath.Dir(goExecutable)

	// Check the current file size in number of bytes
	// TODO: See if this step is faster than monitoring for an EOF during the parsing
	// f, err := os.Stat(filepath.Join(executablePath, "m.csv"))
	f, err := os.Stat("C:\\Users\\Rasky\\Documents\\App_Development\\Golang_Projects\\Billion-Row-Challenge\\measurements.csv")
	if err != nil {
		panic(fmt.Sprintf(">>> - %v", err))
	}
	bytesInFile := f.Size()
	numberOfRoutineCalls := bytesInFile / utilities.BufferSize // Will return an int64 value

	finalBufferSize := bytesInFile - (numberOfRoutineCalls * utilities.BufferSize)

	// file, err := os.Open(filepath.Join(executablePath, "measurements.csv"))
	// file, err := os.Open(filepath.Join(executablePath, "m.csv"))
	file, err := os.Open("C:\\Users\\Rasky\\Documents\\App_Development\\Golang_Projects\\Billion-Row-Challenge\\measurements.csv")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			panic(err)
		}
	}()

	// Create a determined number of routines that will parse the file.
	// TODO: Test if more or less of these impact the speed
	// for range utilities.NumberOfReaderRoutines {
	// for range utilities.NumberOfReaderRoutines {
	// 	go multireader.PartialFileReader(file, utilities.BufferSize, &waitGroup)
	// }

	// Launch a routine that manages the partial reads that will occur throughout the file
	go parsers.PartialReadManager(&waitGroup, numberOfRoutineCalls)

	// Launch a routine that will aggregate all the different rows into the output map
	go output.AggregateEntryOutputs(utilities.OutputMap, &waitGroup)

	// Signal the reader to move through the file and read the specified chunks
	for loopIndex := range int64(numberOfRoutineCalls) {
		multireader.PartialFileReaderNoRoutine(file, utilities.BufferSize, &waitGroup, loopIndex*utilities.BufferSize, loopIndex)
		// multireader.FileReadSectionChannel <- multireader.FileReadSectionFields{
		// 	FileOffset: loopIndex * utilities.BufferSize, // Offset by the buffer size each loop
		// 	Index:      loopIndex,                        // Track which loop the program is currently on
		// }
	}

	// Send the final read to finish parsing all the byte values within the file
	if finalBufferSize > 0 {
		// go multireader.FinalFileReader(
		multireader.FinalFileReader(
			file,
			finalBufferSize,
			numberOfRoutineCalls*utilities.BufferSize,
			int64(numberOfRoutineCalls),
			&waitGroup,
		)
	}

	time.Sleep(time.Millisecond * 125)

	// Wait until all entries have been parsed and placed into the output map
	waitGroup.Wait()

	fmt.Println(utilities.OutputMap)

	// Get the current working directory
	// TODO: Strip this out before the competition and hard-code the path to the file to speed up execution.
	// goExecutable, err := os.Executable()
	// if err != nil {
	// 	panic(err)
	// }
	// executablePath := filepath.Dir(goExecutable)

	// // Check the current file size in number of bytes
	// // TODO: See if this step is faster than monitoring for an EOF during the parsing
	// f, err := os.Stat(filepath.Join(executablePath, "measurements.csv"))
	// if err != nil {
	// 	panic(err)
	// }
	// bytesInFile := f.Size()
	// numberOfRoutineCalls := bytesInFile / utilities.BufferSize // Will return an int64 value

	// finalBufferSize := bytesInFile - (numberOfRoutineCalls * utilities.BufferSize)

	// // file, err := os.Open(filepath.Join(executablePath, "measurements.csv"))
	// file, err := os.Open(filepath.Join(executablePath, "measurements.csv"))
	// if err != nil {
	// 	panic(err)
	// }
	// defer func() {
	// 	if err = file.Close(); err != nil {
	// 		panic(err)
	// 	}
	// }()

	// // Launch a number of routines that will sit and process read buffers as they arrive
	// for _ = range utilities.NumberOfReaderRoutines {
	// 	go multireader.PartialReader(file, utilities.BufferSize)
	// }

	// go parsers.CombinePartialReads()

	// // Start up the channel calls
	// for i := range int64(numberOfRoutineCalls) {
	// 	multireader.ReaderChan <- multireader.ReaderFields{
	// 		Offset: i * utilities.BufferSize,
	// 		Index:  i,
	// 	}

	// 	if i == numberOfRoutineCalls {
	// 		fmt.Println("Basically done")
	// 		// fmt.Println(time.Since(start))
	// 	}
	// }

	// if finalBufferSize > 0 {
	// 	// go multireader.FinalReader(file, finalBufferSize, numberOfRoutineCalls*utilities.BufferSize, int64(numberOfRoutineCalls))
	// 	go multireader.FinalReader(file, finalBufferSize, numberOfRoutineCalls*utilities.BufferSize, int64(numberOfRoutineCalls))
	// }

	// time.Sleep(time.Minute * 5)
}
