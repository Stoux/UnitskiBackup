package unitski

import (
	"fmt"
	"log"
	"os"
	"time"
)

func SetLogger() {
	// TODO: Output logging to input & file
	// Check if the logs folder exists
	if stat, dirErr := os.Stat("logs"); os.IsNotExist(dirErr) {
		if mkDirErr := os.Mkdir("logs", os.ModePerm); mkDirErr != nil {
			panic(mkDirErr)
		}
	} else if !stat.IsDir() {
		panic("logs isn't a folder?")
	}

	// Create the file with the current date as name
	now := time.Now()
	name := fmt.Sprintf("logs/backup-%v.log", now.Format("2006-01-02")) // 2006-01-02_15:04:05
	file, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		// We need that file.
		panic(err)
	}

	// Set it as output of the logger
	log.SetOutput(file)
}
