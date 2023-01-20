package go_ensurefips

import (
	"log"
	"os"
)

// Compliant always returns success on darwin/development machines
func Compliant() {
	logger := log.New(os.Stdout, "go_ensurefips: ", log.Ldate|log.Ltime|log.Llongfile)
	logger.Printf("NOOP FIPS compliance check on darwin")
}
