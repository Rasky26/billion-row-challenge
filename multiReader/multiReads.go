package multireader

import (
	"billionRowChallenge/parsers"
	"billionRowChallenge/utilities"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// ReaderFields - Key fields that come across the channel:
// - Offset: Indicates what section of the file is to be read
// - Index:  Tells the routine which loop value it is on. This will be used to join together partial string values.
type ReaderFields struct {
	Offset int64
	Index  int64
}

// ReaderChan - Singular channel that will act as the controller and feeder of incoming read commands to the different
// running routines.
// TODO: Play around with the buffer size and see if that makes an impact.
var ReaderChan = make(chan ReaderFields, utilities.NumberOfReaderRoutines)

// PartialReader - Ran as a go routine that will have a dedicated buffer the incoming data will be streamed in. Because multiple
// routines will all be reading from the same channel, it will be a race condition which (hopefully) should help minimize downtime.
func PartialReader(file *os.File, bufferSize int64) {

	// Set a consistent buffer that will last through the entirety of the go routine running.
	var buffer = make([]byte, 100)

	// Loop forever? Or should I work on closing out these go routines once the reading process is done?
	for {

		// First one to grab from the channel wins!
		request := <-ReaderChan

		// Move the reader to the offset value and read in the specified number of bytes
		reader := io.NewSectionReader(file, request.Offset, bufferSize)
		n, err := reader.Read(buffer)

		// TODO: Probably remove this error checking during the competition to speed up execution. It's bad to do, but speed wins here!
		if errors.Is(err, io.EOF) {
			if n < 1 {
				panic(errors.New("some error"))
			}
			fmt.Println(request.Index, "ERROR:", string(buffer[:n]))
			panic(errors.New("some error 2"))
		} else if err != nil {
			panic(err)
		}

		// go parsers.ParseBytes(buffer, request.Index)
		parsers.ParseBytes(buffer, request.Index)
	}
}

// PartialReader - Ran as a go routine that will have a dedicated buffer the incoming data will be streamed in. Because multiple
// routines will all be reading from the same channel, it will be a race condition which (hopefully) should help minimize downtime.
func FinalReader(file *os.File, bufferSize int64, offset int64, index int64) {

	// Set a consistent buffer that will last through the entirety of the go routine running.
	buffer := make([]byte, bufferSize)

	// Move the reader to the offset value and read in the specified number of bytes
	reader := io.NewSectionReader(file, offset, bufferSize)
	n, err := reader.Read(buffer)

	// TODO: Probably remove this error checking during the competition to speed up execution. It's bad to do, but speed wins here!
	if errors.Is(err, io.EOF) {
		if n < 1 {
			panic(err)
		}
		fmt.Println(index, "ERROR:", string(buffer[:n]))
		panic(err)
	} else if err != nil {
		panic(err)
	}

	go parsers.ParseBytes(buffer, index)
}

// TODO: See if string to int conversion is quicker. Or if I make a map of all the potential values and loop up those conversions there.

// ============================ NEW STUFF ===================================
// FileReadSectionFields - Fields that indicate what section of the file should be read and what the index of the read
// is. The index is used to link together partial data read from the file.
type FileReadSectionFields struct {
	FileOffset int64
	Index      int64
}

// FileReadSectionChannel - Channel that will accept the incoming byte reads, including the starting offset
// and the set index of the read
var FileReadSectionChannel = make(chan FileReadSectionFields)

// PartialFileReader - Will read chunks out of the specified file and sends that data off for further processing
func PartialFileReader(file *os.File, bufferSize int64, entryWaitGroup *sync.WaitGroup) {

	// Set a consistent buffer that will last through the entirety of the go routine running.
	var readBuffer = make([]byte, bufferSize)

	// Loop forever
	for {

		// Listen for new read requests
		readTarget := <-FileReadSectionChannel

		// Move the reader to the offset value and read in the specified number of bytes
		reader := io.NewSectionReader(file, readTarget.FileOffset, bufferSize)

		// Put those byte values into the defined static buffer
		n, err := reader.Read(readBuffer)

		// TODO: Probably remove this error checking during the competition to speed up execution. It's bad to do, but speed wins here!
		if errors.Is(err, io.EOF) {
			if n < 1 {
				panic(errors.New("some error"))
			}
			fmt.Println(readTarget.Index, "ERROR:", string(readBuffer[:n]))
			panic(errors.New("some error 2"))
		} else if err != nil {
			panic(err)
		}

		// Send the buffer of bytes values off to be processed
		go parsers.ParseByteBuffer(readBuffer, readTarget.Index, entryWaitGroup)
	}
}

// FinalFileReader - Routine that simply manages the final read out of the file
func FinalFileReader(file *os.File, bufferSize int64, offset int64, index int64, entryWaitGroup *sync.WaitGroup) {

	// Create a buffer exactly equal to the last number of bytes that need to be read out of the file
	var readBuffer = make([]byte, bufferSize)

	// Move the reader to the offset value and read in the specified number of bytes
	reader := io.NewSectionReader(file, offset, bufferSize)

	// Put those byte values into the defined static buffer
	n, err := reader.Read(readBuffer)

	// TODO: Probably remove this error checking during the competition to speed up execution. It's bad to do, but speed wins here!
	if errors.Is(err, io.EOF) {
		if n < 1 {
			panic(err)
		}
		fmt.Println(index, "ERROR:", string(readBuffer[:n]))
		panic(err)
	} else if err != nil {
		panic(err)
	}

	// Fire off a routine to inspect the final values
	go parsers.ParseByteBuffer(readBuffer, index, entryWaitGroup)
}
