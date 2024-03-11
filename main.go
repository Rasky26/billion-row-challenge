package main

import (
	"billionRowChallenge/expectedOutput"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	fmt.Println("Heyo!")

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)

	output := expectedOutput.CalculateExpectedOutput(filepath.Join(exPath, "measurements.csv"))

	file, err := os.Create("result.txt")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			panic(err)
		}
	}()

	_, err = file.Write([]byte(output))
	if err != nil {
		panic(err)
	}
}
