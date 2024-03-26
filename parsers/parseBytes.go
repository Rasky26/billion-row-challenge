package parsers

import (
	"billionRowChallenge/output"
	"billionRowChallenge/utilities"
	"fmt"
	"strconv"
	"sync"
)

type PartialReadFields struct {
	index            int64
	city             []byte
	temperatureWhole []byte
	decimalPoint     byte
}

var PartialReadChannel = make(chan PartialReadFields)

func ParseBytes(byteData []byte, mainIndex int64) {

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
			PartialReadChannel <- PartialReadFields{
				index:            mainIndex,
				city:             cityByteSlice,
				temperatureWhole: temperatureWholeByteSlice,
				decimalPoint:     temperatureDecimalByte,
			}

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
				go parseEntry(
					byteData[byteSliceStartingIndex:byteFields[utilities.SemiColonIndex].index],
					byteData[byteFields[utilities.SemiColonIndex].index+1:byteFields[utilities.DecimalIndex].index],
					byteData[byteFields[utilities.NewLineIndex].index-1],
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
	PartialReadChannel <- PartialReadFields{
		index:            mainIndex + 1, // Use `+1` as the very first and last row of data will never need to be linked
		city:             cityByteSlice,
		temperatureWhole: temperatureWholeByteSlice,
		decimalPoint:     temperatureDecimalByte,
	}
}

func parseEntry(cityByteSlice []byte, temperatureWholeByteSlice []byte, temperatureDecimalByte byte) {

	// temperature := append(temperatureWholeByteSlice, temperatureDecimalByte)

	// fmt.Println("parseEntry:", string(cityByteSlice), string(temperature))
}

func CombinePartialReads() {

	type CombinedReadFields struct {
		city             []byte
		temperatureWhole []byte
		decimalPoint     byte
	}
	var partialReadMap = make(map[int64]CombinedReadFields)

	for {
		partialRead := <-PartialReadChannel

		if partialRead.index == 0 {
			// fmt.Println("FIRST: City:", string(partialRead.city), " - Temp:", string(append(partialRead.temperatureWhole, partialRead.decimalPoint)))
			continue
		}

		value, ok := partialReadMap[partialRead.index]
		if !ok {
			partialReadMap[partialRead.index] = CombinedReadFields{
				city:             partialRead.city,
				temperatureWhole: partialRead.temperatureWhole,
				decimalPoint:     partialRead.decimalPoint,
			}
		} else {

			var city []byte
			var temperature []byte

			// Need to determine which of the ordering to combine the fields together
			//
			// If the decimal field was set to `nil` manually, then the `value` field contains the leading information
			// value:       `[]byte{'CityN'} []byte 0x00`
			// partialRead: `[]byte{ame} []byte{26} 0x2`
			if value.decimalPoint == 0x00 {
				city = append(value.city, partialRead.city...)
				temperature = append(value.temperatureWhole, partialRead.temperatureWhole...)
				temperature = append(temperature, partialRead.decimalPoint)
			} else {
				city = append(partialRead.city, value.city...)
				temperature = append(partialRead.temperatureWhole, value.temperatureWhole...)
				temperature = append(temperature, value.decimalPoint)
			}

			if len(city) < 1 {
				fmt.Println(city)
			}

			delete(partialReadMap, partialRead.index)

			// fmt.Println("Combination: City:", string(city), " - Temp:", string(temperature))
		}
	}
}

// // ============ NEW STUFF ========================
// PartialEntryFields - Holds the byte values that were partially parsed from the main file
type PartialEntryFields struct {
	City             []byte
	TemperatureField []byte
	DecimalField     byte
}
type PartialEntryFieldsString struct { // FIX
	City             string
	TemperatureField string
	DecimalField     string
}

// PartialEntryMap - Holds the partial entries that were parsed out of the main file. Holds the values on the key
// value of the index from the loop. This index will link together which loop the partial read was parsed from and allow
// for a quick association of those partial fields back into a whole field.
var PartialEntryMap = make(map[int64]PartialEntryFieldsString)

// PartialReadByteFields - Contains the partial information that was read from the file. Includes the index of the read, that
// index will be used to link together partial reads.
type PartialReadByteFields struct {
	Index            int64
	City             []byte
	TemperatureWhole []byte
	DecimalPoint     byte
}
type PartialReadByteFieldsString struct {
	Index            int64
	City             string
	TemperatureWhole string
	DecimalPoint     string
}

// PartialReadChannel2 - Channel that will take in a partial read and either add it to a holding channel
// or match that partial read with an existing partial read to create a new output entry
var PartialReadChannel2 = make(chan PartialReadByteFieldsString)

// ParseByteBuffer - Routine that will take in a buffer from the file and begin parsing the entries to split apart
// the city, the whole temperature value, and the temperature decimal field.
//
// Whole entries will be send for quick processing and be added to the output map
// Partial entries will be send to a partial entry manger that will aggregate other partial entries and re-construct those
// fields into a whole value.
func ParseByteBuffer(byteData []byte, mainIndex int64, entryWaitGroup *sync.WaitGroup) {

	var headerOffset int                 // Indicates where the first full byte slice of values exists
	var byteSliceStartingIndex int       // Starting index of the current line
	var targetByteToCheckFor uint        // Rotate which byte character is currently being watched for. Start with the newline code, as that will then start the rotating key process
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

	// if strings.Contains(string(incomingByteData), "gzhou;") {
	// 	fmt.Println(string(incomingByteData))
	// }

	// var byteData = make([]byte, utilities.BufferSize)
	// n := copy(byteData, incomingByteData)
	// fmt.Printf("%v %v %v\n", n, len(byteData), len(incomingByteData))

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

			headerOffset = index + 1           // Used to move the header for the array when looping below
			byteSliceStartingIndex = index + 1 // Indicates where the next valid row of data will begin

			// fmt.Printf("Initial Line:\n\tIndex: %v\n\tSemicolon: %v\n\tDecimal: %v\n\tNewLine: %v\n\tCity: %v\n\tTemperature: %v\n\tOffset: %v\n\n",
			// 	mainIndex,
			// 	byteFields[utilities.SemiColonIndex].index,
			// 	byteFields[utilities.DecimalIndex].index,
			// 	byteFields[utilities.NewLineIndex].index,
			// 	string(cityByteSlice),
			// 	string(append(temperatureWholeByteSlice, temperatureDecimalByte)),
			// 	headerOffset,
			// )

			// Send the partial byte arrays over to be stored and linked together. This will usually contain
			// only the ending values of the partial read and the `temperatureDecimalByte` value should usually
			// be a value other than `0x00`.
			PartialReadChannel2 <- PartialReadByteFieldsString{
				Index:            mainIndex,
				City:             string(cityByteSlice),
				TemperatureWhole: string(temperatureWholeByteSlice),
				DecimalPoint:     string(temperatureDecimalByte),
			}

			// Exit this loop. All other entries (other than the final entry) will contain complete data.
			break
		}
	}

	// Loop over the byte slice
	// TODO: Do another multi-read of the byte slice
	for index := range byteData[headerOffset:] {

		// Because the appear of the target byte fields (`;`, `.`, `\n`) always appear in the same order,
		// can reduce this `if` check down to a single evaluation per byte inspection.
		if byteData[headerOffset:][index] == byteFields[targetByteToCheckFor].byteValue {
			byteFields[targetByteToCheckFor].index = index + headerOffset

			// Move to the next target byte to inspect
			targetByteToCheckFor++

			// Once a newline character is found, signal that the full byte slice is set and reset
			// the target byte for inspection back to the semicolon symbol
			if targetByteToCheckFor > 2 {

				// fmt.Printf("Regular Line:\n\tIndex: %v\n\tSemicolon: %v\n\tDecimal: %v\n\tNewLine: %v\n\tCity: %v\n\tTemperature: %v\n\tOffset: %v\n\n",
				// 	mainIndex,
				// 	byteFields[utilities.SemiColonIndex].index,
				// 	byteFields[utilities.DecimalIndex].index,
				// 	byteFields[utilities.NewLineIndex].index,
				// 	string(byteData[byteSliceStartingIndex:byteFields[utilities.SemiColonIndex].index]),
				// 	string(append(byteData[byteFields[utilities.SemiColonIndex].index+1:byteFields[utilities.DecimalIndex].index], byteData[byteFields[utilities.NewLineIndex].index-1])),
				// 	headerOffset,
				// )

				// A full byte slice has been found and can be parsed
				go ParseCompleteEntry(
					byteData[byteSliceStartingIndex:byteFields[utilities.SemiColonIndex].index],
					byteData[byteFields[utilities.SemiColonIndex].index+1:byteFields[utilities.DecimalIndex].index],
					byteData[byteFields[utilities.NewLineIndex].index-1],
					entryWaitGroup,
				)

				targetByteToCheckFor = 0                          // Reset the inspector for the next loop
				byteSliceStartingIndex = index + headerOffset + 1 // Set the starting index for the next byte slice
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

	// fmt.Printf("Last Line:\n\tIndex: %v\n\tSemicolon: %v\n\tDecimal: %v\n\tNewLine: %v\n\tCity: %v\n\tTemperature: %v\n\tOffset: %v\n\n",
	// 	mainIndex,
	// 	byteFields[utilities.SemiColonIndex].index,
	// 	byteFields[utilities.DecimalIndex].index,
	// 	byteFields[utilities.NewLineIndex].index,
	// 	string(cityByteSlice),
	// 	string(append(temperatureWholeByteSlice, temperatureDecimalByte)),
	// 	headerOffset,
	// )

	// Send the partial byte arrays over to be stored and linked together. This will usually contain
	// only the starting values of the partial read and the `temperatureDecimalByte` value should usually
	// be `0x00`, indicating to the aggregator that this data is the prefix values.
	PartialReadChannel2 <- PartialReadByteFieldsString{
		Index:            mainIndex + 1, // Increment by one, to make sure the aggregator can link these leading partial values with the next trailing partial values
		City:             string(cityByteSlice),
		TemperatureWhole: string(temperatureWholeByteSlice),
		DecimalPoint:     string(temperatureDecimalByte),
	}
}

// ParseCompleteEntry - Routine that accepts the incoming byte values, parses those values into the expected
// output format, and sends it off to the aggregator that will add it to the output map
func ParseCompleteEntry(cityByteSlice []byte, temperatureWholeByteSlice []byte, temperatureDecimalByte byte, entryWaitGroup *sync.WaitGroup) {

	// Add to the current wait group
	entryWaitGroup.Add(1)

	// Combine the temperature byte arrays into a singular byte array
	temperature := append(temperatureWholeByteSlice, temperatureDecimalByte)

	// Convert the temperature byte values into an integer, multiplied by 10
	temperatureValue, err := strconv.Atoi(string(temperature))
	if err != nil {
		panic(err)
	}

	// Send the valid output to the aggregator to be added to the output map
	output.OutputEntryChannel <- output.OutputEntry{
		City:        string(cityByteSlice),
		Temperature: temperatureValue,
	}
}

// PartialReadManager - Manages the partial reads that occur when reading chucks of byte data from the file.
// Links together partial data by utilizing the index value from the read to determine which fields need
// to be linked together. Once a field is fully linked together, send the information to the map to be aggregated
// into the results.
func PartialReadManager(entryWaitGroup *sync.WaitGroup, numberOfRoutineCalls int64) {

	// var cityCompleteArray []byte        // Contains the combined partial reads and creates a full city name
	// var temperatureCompleteArray []byte // Contains the combined partial temperature reads and creates a complete temperature entry
	var cityCompleteArray string        // FIX
	var temperatureCompleteArray string // FIX

	// Loop forever
	for {

		// Await for incoming partial reads
		partialEntry := <-PartialReadChannel2

		// Need a special case for the very first entry read from the file. Don't love this, but it's the best we have currently.
		if partialEntry.Index == 0 {

			// Add a wait group for this first entry
			entryWaitGroup.Add(1)

			// Combine the whole temperature value with the decimal value into a singular array
			// temperatureCompleteArray := append(partialEntry.TemperatureWhole, partialEntry.DecimalPoint)
			temperatureCompleteArray := partialEntry.TemperatureWhole + partialEntry.DecimalPoint // FIX

			// Convert the byte array to the temperature equivalent, multiplied by 10
			temperatureValue, err := strconv.Atoi(string(temperatureCompleteArray))
			if err != nil {
				panic(err)
			}

			// Send the output to the aggregation channel for processing
			output.OutputEntryChannel <- output.OutputEntry{
				City:        string(partialEntry.City),
				Temperature: temperatureValue,
			}

			continue
		}

		if partialEntry.Index+1 > numberOfRoutineCalls {
			continue
		}

		// Locate any existing partial entry within the map
		value, ok := PartialEntryMap[partialEntry.Index]

		// Create a new entry that will hold the partial values and store it until the matching partial entry is found
		if !ok {
			PartialEntryMap[partialEntry.Index] = PartialEntryFieldsString{
				City:             partialEntry.City,
				TemperatureField: partialEntry.TemperatureWhole,
				DecimalField:     partialEntry.DecimalPoint,
			}

			// Move to the next loop and await more information
			continue
		}

		// Add a wait group as this will be the second half of the partial value and the data will be combined and parsed
		entryWaitGroup.Add(1)

		// Need to determine which of the ordering to combine the fields together
		//
		// If the decimal field was set to `nil` manually, then the `value` field contains the leading information
		// value:       `[]byte{'CityN'} []byte 0x00`
		// partialRead: `[]byte{ame} []byte{26} 0x2`
		//
		// If the existing entry has a `nil` decimal field, then that entry STARTS with the valid city bytes
		if value.DecimalField == string(0x00) {
			// cityCompleteArray = append(value.City, partialEntry.City...)
			// temperatureCompleteArray = append(value.TemperatureField, partialEntry.TemperatureWhole...)
			// temperatureCompleteArray = append(temperatureCompleteArray, partialEntry.DecimalPoint)
			cityCompleteArray = value.City + partialEntry.City // FIX
			temperatureCompleteArray = value.TemperatureField + partialEntry.TemperatureWhole
			temperatureCompleteArray = temperatureCompleteArray + partialEntry.DecimalPoint
		} else {

			// Otherwise, the existing entry contains the ENDING byte values, and the incoming information should prepend
			// it's information
			// cityCompleteArray = append(partialEntry.City, value.City...)
			// temperatureCompleteArray = append(partialEntry.TemperatureWhole, value.TemperatureField...)
			// temperatureCompleteArray = append(temperatureCompleteArray, value.DecimalField)
			cityCompleteArray = partialEntry.City + value.City // FIX
			temperatureCompleteArray = partialEntry.TemperatureWhole + value.TemperatureField
			temperatureCompleteArray = temperatureCompleteArray + value.DecimalField
		}

		// Convert the temperature byte array into the integer value, multiplied by 10
		temperatureValue, err := strconv.Atoi(string(temperatureCompleteArray))
		if err != nil {
			panic(err)
		}

		// Send off the complete row entry to be added into the output map
		output.OutputEntryChannel <- output.OutputEntry{
			City:        string(cityCompleteArray),
			Temperature: temperatureValue,
		}

		// To keep my sad little computer from starting on fire...
		delete(PartialEntryMap, partialEntry.Index)
	}
}
