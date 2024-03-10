package expectedOutput

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
)

type outputFields struct {
	minTemp      int
	maxTemp      int
	runningTotal int
	count        int
}

// CalculateExpectedOutput - VERY SLOW!!! Finds the expected output from the billion rows.
// Just does a basic loop and finds the output. Puts that output into a file called `answer.txt`.
// Used to validate future builds against.
func CalculateExpectedOutput(filename string) string {

	var cityTemperatures = make(map[string]outputFields)

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			panic(err)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		processRow(scanner.Text(), cityTemperatures)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	keyMap := make([]string, 0, len(cityTemperatures))

	for key := range cityTemperatures {
		keyMap = append(keyMap, key)
	}

	slices.Sort(keyMap)

	answer := "{"
	for _, key := range keyMap {
		answer += fmt.Sprintf(
			"%v=%.1f/%.1f/%.1f, ",
			key,
			float64(cityTemperatures[key].minTemp)/10,
			float64(cityTemperatures[key].runningTotal)/float64(cityTemperatures[key].count)/10,
			float64(cityTemperatures[key].maxTemp)/10,
		)
	}
	answer = answer[:len(answer)-2] + "}"

	return answer
}

type weatherFields struct {
	station     string
	temperature int
}

func processRow(fields string, cityTemperatures map[string]outputFields) {

	splitString := strings.Split(fields, ";")
	temperature, err := strconv.Atoi(strings.ReplaceAll(splitString[1], ".", ""))
	if err != nil {
		panic(err)
	}

	wxField := weatherFields{
		station:     strings.TrimSpace(splitString[0]),
		temperature: temperature,
	}

	field, ok := cityTemperatures[wxField.station]
	if ok {
		if wxField.temperature > field.maxTemp {
			field.maxTemp = wxField.temperature
		} else if wxField.temperature < field.minTemp {
			field.minTemp = wxField.temperature
		}

		field.runningTotal += wxField.temperature
		field.count++
	} else {
		field.maxTemp = wxField.temperature
		field.minTemp = wxField.temperature
		field.runningTotal = wxField.temperature
		field.count = 1
	}

	cityTemperatures[wxField.station] = field
}
