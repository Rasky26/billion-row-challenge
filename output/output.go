package output

import (
	"billionRowChallenge/utilities"
	"sync"
)

// OutputEntry - Fields that are sent to the output map
type OutputEntry struct {
	City        string
	Temperature int
}

// OutputEntryChannel - Channel that aggregates the output values and will move those entries into the output map
var OutputEntryChannel = make(chan OutputEntry)

// AggregateEntryOutputs - Routine that will listen for incoming row entries of city and temperature fields
func AggregateEntryOutputs(outputMap map[string]utilities.OutputValues, entryWaitGroup *sync.WaitGroup) {

	for {

		// Listen for incoming map update calls. This only processes one at a time, but maybe look at
		// creating a few different versions of this and then doing a final aggregation.
		rowEntry := <-OutputEntryChannel

		// Locate any existing record
		mapEntry, ok := outputMap[rowEntry.City]

		// No entry exists, then create a new entry into the output map
		if !ok {

			outputMap[rowEntry.City] = utilities.OutputValues{
				Min:   rowEntry.Temperature,
				Max:   rowEntry.Temperature,
				Total: rowEntry.Temperature,
				Count: 1,
			}

			entryWaitGroup.Done()

			// Move to the next loop and listen for an output
			continue
		}

		// Update the values to track the min, max, and total counts
		if mapEntry.Min > rowEntry.Temperature {
			mapEntry.Min = rowEntry.Temperature
		} else if mapEntry.Max < rowEntry.Temperature {
			mapEntry.Max = rowEntry.Temperature
		}
		mapEntry.Total += rowEntry.Temperature
		mapEntry.Count++

		// Update the map with the latest values
		outputMap[rowEntry.City] = mapEntry

		// Remove a wait group
		entryWaitGroup.Done()
	}
}
