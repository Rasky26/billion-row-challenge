package parsers

import (
	"billionRowChallenge/utilities"
	"fmt"
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

	temperature := append(temperatureWholeByteSlice, temperatureDecimalByte)

	fmt.Println("parseEntry:", string(cityByteSlice), string(temperature))
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
			fmt.Println("FIRST: City:", string(partialRead.city), " - Temp:", string(append(partialRead.temperatureWhole, partialRead.decimalPoint)))
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

			delete(partialReadMap, partialRead.index)

			fmt.Println("Combination: City:", string(city), " - Temp:", string(temperature))
		}
	}
}
