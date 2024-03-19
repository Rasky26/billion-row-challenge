package utilities

const BufferSize int64 = 100 // The amount of bytes that are read into the go routine buffer with each read

const NumberOfReaderRoutines = 1 // The amount of go routines that will be created and ready to read data processed through the reader

const NewLineHex = 0xa
const SemicolonHex = 0x3b
const DecimalHex = 0x2e
const SemiColonIndex = 0
const DecimalIndex = 1
const NewLineIndex = 2

type OutputValues struct {
	Min   int
	Max   int
	Total int
	Count int
}

var OutputMap = make(map[string]OutputValues)
