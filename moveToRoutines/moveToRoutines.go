package movetoroutines

import (
	"billionRowChallenge/utilities"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

type outputValues struct {
	min   int
	max   int
	total int
	count int
}

var output = make(map[string]outputValues)

// main - Core entry point to the Billion Row Challenge
func BuildRoutinesMain() {

	// var parserWaitGroup sync.WaitGroup
	var entryWaitGroup sync.WaitGroup

	// Get the current working directory
	// TODO: Strip this out before the competition and hard-code the path to the file to speed up execution.
	goExecutable, err := os.Executable()
	if err != nil {
		panic(err)
	}
	executablePath := filepath.Dir(goExecutable)

	// Check the current file size in number of bytes
	// TODO: See if this step is faster than monitoring for an EOF during the parsing
	f, err := os.Stat(filepath.Join(executablePath, "m.csv"))
	if err != nil {
		panic(err)
	}
	bytesInFile := f.Size()
	numberOfRoutineCalls := bytesInFile / utilities.BufferSize // Will return an int64 value

	finalBufferSize := bytesInFile - (numberOfRoutineCalls * utilities.BufferSize)

	// file, err := os.Open(filepath.Join(executablePath, "measurements.csv"))
	file, err := os.Open(filepath.Join(executablePath, "m.csv"))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			panic(err)
		}
	}()

	// Start up the channel calls
	for i := range int64(numberOfRoutineCalls) {
		partialReader(file, utilities.BufferSize, i*utilities.BufferSize, i, &entryWaitGroup)
	}

	if finalBufferSize > 0 {
		finalReader(file, finalBufferSize, numberOfRoutineCalls*utilities.BufferSize, int64(numberOfRoutineCalls), &entryWaitGroup)
	}

	fmt.Println(output)

	// parserWaitGroup.Wait()
	entryWaitGroup.Wait()
}

func partialReader(file *os.File, bufferSize int64, offset int64, index int64, entryWaitGroup *sync.WaitGroup) {

	// Set a consistent buffer that will last through the entirety of the go routine running.
	var buffer = make([]byte, 100)

	// Move the reader to the offset value and read in the specified number of bytes
	reader := io.NewSectionReader(file, offset, bufferSize)
	n, err := reader.Read(buffer)

	// TODO: Probably remove this error checking during the competition to speed up execution. It's bad to do, but speed wins here!
	if errors.Is(err, io.EOF) {
		if n < 1 {
			panic(errors.New("some error"))
		}
		fmt.Println(index, "ERROR:", string(buffer[:n]))
		panic(errors.New("some error 2"))
	} else if err != nil {
		panic(err)
	}

	parseBytes(buffer, index, entryWaitGroup)
}

func finalReader(file *os.File, bufferSize int64, offset int64, index int64, entryWaitGroup *sync.WaitGroup) {

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

	parseBytes(buffer, index, entryWaitGroup)
}

// =======================================
func parseBytes(byteData []byte, mainIndex int64, entryWaitGroup *sync.WaitGroup) {

	var initialSliceOffset int           // Indicates where the first full byte slice of values exists
	var byteSliceStartingIndex int       // Starting index of the current line
	var targetByte uint                  // Rotate which byte character is currently being watched for. Start with the newline code, as that will then start the rotating key process
	var cityByteSlice []byte             // Holds the city name
	var temperatureWholeByteSlice []byte // Holds the whole-number value the temperature
	var temperatureDecimalByte byte      // Holds the singular decimal field

	// Hold information about the position of target values
	type byteFieldsStruct struct {
		byteValue byte
		index     int
	}
	// Target values to inspect for and then log the index position of
	var byteFields = [3]byteFieldsStruct{
		{byteValue: utilities.SemicolonHex},
		{byteValue: utilities.DecimalHex},
		{byteValue: utilities.NewLineHex},
	}

	// Read the initial bytes until a newline character is reached. This first slice will require its own processing
	// to link with any slice values found at the trailing end of a different reader.
	for index := range byteData {

		// Check if the current byte is a semicolon
		if byteData[index] == utilities.SemicolonHex {
			byteFields[utilities.SemiColonIndex].index = index

			// If a valid, partial city name is found, set that data
			if index > 0 {
				cityByteSlice = byteData[:index]
			}

		} else if byteData[index] == utilities.DecimalHex {
			// Otherwise, check if the byte is a decimal
			byteFields[utilities.DecimalIndex].index = index

			if index > 0 && byteFields[utilities.SemiColonIndex].index > 0 {
				// If a decimal is found and a semicolon was found after a partial city name, set the target data range.
				// e.g. `yName;26.2\n`
				temperatureWholeByteSlice = byteData[byteFields[utilities.SemiColonIndex].index+1 : index]

			} else if index > 0 && byteData[utilities.SemiColonIndex] == utilities.SemicolonHex {
				// If a decimal is found and the original slice started with a semicolon, set the target data range.
				// e.g. `;26.2\n`
				temperatureWholeByteSlice = byteData[1:index]

			} else if index > 0 {
				// Otherwise, set the starting data as the number before the decimal point.
				// e.g. `6.2\n` --or-- `.2\n`
				temperatureWholeByteSlice = byteData[:index]
			}

		} else if byteData[index] == utilities.NewLineHex {
			if index > 0 {
				// If a newline is found and the index was greater than zero, set the decimal value
				temperatureDecimalByte = byteData[index-1]
			}

			// Set the starting index for the next loop
			initialSliceOffset = index + 1
			byteSliceStartingIndex = index + 1

			// Send the partial data for processing
			combinePartialReads(mainIndex, cityByteSlice, temperatureWholeByteSlice, temperatureDecimalByte, entryWaitGroup)

			// Exit this loop
			break
		}
	}

	// Loop over the byte slice
	// TODO: Do another multi-read of the byte slice
	for index := range byteData[initialSliceOffset:] {

		// Because the appear of the target byte fields (`;`, `.`, `\n`) always appear in the same order,
		// can reduce this `if` check down to a single evaluation per byte inspection.
		if byteData[initialSliceOffset:][index] == byteFields[targetByte].byteValue {
			byteFields[targetByte].index = index + initialSliceOffset

			// Move to the next target byte to inspect
			targetByte++

			// Once a newline character is found, signal that the full byte slice is set and reset
			// the target byte for inspection back to the semicolon symbol
			if targetByte > 2 {

				// A full byte slice has been found and can be parsed
				parseEntry(
					byteData[byteSliceStartingIndex:byteFields[utilities.SemiColonIndex].index],
					byteData[byteFields[utilities.SemiColonIndex].index+1:byteFields[utilities.DecimalIndex].index],
					byteData[byteFields[utilities.NewLineIndex].index-1],
					entryWaitGroup,
				)

				// Reset the inspector for the next loop
				targetByte = 0

				// Set the starting index for the next byte slice
				byteSliceStartingIndex = index + initialSliceOffset + 1
			}
		}
	}

	// ======================================
	// Ending section, so handle partial byte reads and send the data off to be linked with other partial reads
	// ======================================
	if byteFields[utilities.SemiColonIndex].index > byteSliceStartingIndex {
		// This section will ALWAYS start with either the city, or it will be blank.
		// * e.g. `CityName;26.` --or-- `CityNa` --or-- ``
		//
		// First, check if a semicolon was found
		cityByteSlice = byteData[byteSliceStartingIndex:byteFields[utilities.SemiColonIndex].index]

	} else {
		// Otherwise, grab whatever data is available and assign it to the city. This should be a blank array
		// * e.g. ``
		cityByteSlice = byteData[byteSliceStartingIndex:]
	}

	if byteFields[utilities.DecimalIndex].index > byteSliceStartingIndex {
		// Check if a decimal field has been found after the city name
		// * e.g. `CityName;26.`
		temperatureWholeByteSlice = byteData[byteFields[utilities.SemiColonIndex].index+1 : byteFields[utilities.DecimalIndex].index]

	} else if byteFields[utilities.SemiColonIndex].index > byteSliceStartingIndex {
		// Otherwise, if a semicolon was found, grab all the trailing data as the temperature
		// * e.g. `CityName;26` --or-- `CityName;2`
		temperatureWholeByteSlice = byteData[byteFields[utilities.SemiColonIndex].index+1:]

	} else {
		// Finally, if no temperature value is present, then set it to a blank byte array
		// * e.g. `CityName;`
		temperatureWholeByteSlice = []byte{}
	}

	if byteFields[utilities.NewLineIndex].index > byteSliceStartingIndex {
		// Check if the end of the data ends with a newline character
		// * e.g. `CityName;26.2\n`
		temperatureDecimalByte = byteData[byteFields[utilities.NewLineIndex].index-1]

	} else if byteFields[utilities.DecimalIndex].index > byteSliceStartingIndex && byteFields[utilities.DecimalIndex].index < len(byteData)-1 {
		// Otherwise, if a decimal was found, check if the decimal index was NOT the last value. Meaning, the last
		// value is actually the decimal field.
		// * e.g. `CityName;26.2`
		temperatureDecimalByte = byteData[byteFields[utilities.DecimalIndex].index+1]

	} else {
		// Finally, if not decimal value is present, then set it to a blank byte value
		temperatureDecimalByte = 0x00 // Set it to a `nil`, to indicate no value was found
	}

	// Send the partial data for processing
	combinePartialReads(mainIndex+1, cityByteSlice, temperatureWholeByteSlice, temperatureDecimalByte, entryWaitGroup)
}

func parseEntry(cityByteSlice []byte, temperatureWholeByteSlice []byte, temperatureDecimalByte byte, entryWaitGroup *sync.WaitGroup) {

	entryWaitGroup.Add(1)

	temperature := append(temperatureWholeByteSlice, temperatureDecimalByte)

	temperatureValue, err := strconv.Atoi(string(temperature))
	if err != nil {
		panic(err)
	}

	addOutput(string(cityByteSlice), temperatureValue, entryWaitGroup)
}

type CombinedReadFields struct {
	city             []byte
	temperatureWhole []byte
	decimalPoint     byte
}

var partialReadMap = make(map[int64]CombinedReadFields)

func combinePartialReads(index int64, city []byte, temperatureWhole []byte, decimalPoint byte, entryWaitGroup *sync.WaitGroup) {

	// type CombinedReadFields struct {
	// 	city             []byte
	// 	temperatureWhole []byte
	// 	decimalPoint     byte
	// }
	// var partialReadMap = make(map[int64]CombinedReadFields)

	if index == 0 {

		// Add a wait group
		entryWaitGroup.Add(1)

		temperature := append(temperatureWhole, decimalPoint)

		temperatureValue, err := strconv.Atoi(string(temperature))
		if err != nil {
			panic(err)
		}

		addOutput(string(city), temperatureValue, entryWaitGroup)
		// fmt.Println("FIRST: City:", string(city), "- Temp:", string(append(temperatureWhole, decimalPoint)))
		return
	}

	value, ok := partialReadMap[index]
	if !ok {
		partialReadMap[index] = CombinedReadFields{
			city:             city,
			temperatureWhole: temperatureWhole,
			decimalPoint:     decimalPoint,
		}
	} else {

		// Add a wait group
		entryWaitGroup.Add(1)

		var cityOutput []byte
		var temperature []byte

		// Need to determine which of the ordering to combine the fields together
		//
		// If the decimal field was set to `nil` manually, then the `value` field contains the leading information
		// value:       `[]byte{'CityN'} []byte 0x00`
		// partialRead: `[]byte{ame} []byte{26} 0x2`
		if value.decimalPoint == 0x00 {
			cityOutput = append(value.city, city...)
			temperature = append(value.temperatureWhole, temperatureWhole...)
			temperature = append(temperature, decimalPoint)
		} else {
			cityOutput = append(city, value.city...)
			temperature = append(temperatureWhole, value.temperatureWhole...)
			temperature = append(temperature, value.decimalPoint)
		}

		temperatureValue, err := strconv.Atoi(string(temperature))
		if err != nil {
			panic(err)
		}

		delete(partialReadMap, index)

		addOutput(string(cityOutput), temperatureValue, entryWaitGroup)
		// fmt.Println("Combination: City:", string(cityOutput), "- Temp:", string(temperature))
	}
}

func addOutput(city string, temperature int, entryWaitGroup *sync.WaitGroup) {

	value, ok := output[city]
	if !ok {

		output[city] = outputValues{
			min:   temperature,
			max:   temperature,
			total: temperature,
			count: 1,
		}
	} else {
		if value.min > temperature {
			value.min = temperature
		} else if value.max < temperature {
			value.max = temperature
		}
		value.total += temperature
		value.count++

		output[city] = value
	}

	// Remove a wait group
	entryWaitGroup.Done()
}
