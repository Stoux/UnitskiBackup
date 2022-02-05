package commands

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"log"
	"time"
	"unitski-backup/unitski"
)

func Sync(configFilePath string) {
	fmt.Println("Running...")
	unitski.SetLogger()
	log.Println("---- Starting backup routine")

	// Load config
	config := unitski.LoadConfig(configFilePath)
	log.Println("Databases", len(config.Databases))

	cli, ctx := unitski.InitDocker()

	databases(cli, ctx, config)
	files(config)

	// TODO: Async
	// Check if required commands are available
	// Start 2 threads: 1 for processing backups, 1 for syncing to backup

	fmt.Println("Done.")
}

func databases(cli *client.Client, ctx context.Context, config unitski.BackupConfig) {
	date := time.Now().Format("2006-01-02")

	// Loop through each database
	for _, database := range config.Databases {
		if !database.Enabled {
			log.Println("Skipping backup of database: " + database.Name + " (is disabled)")
			continue
		}

		log.Println("Starting backup of database: " + database.Name)

		// Create the project folder if not done yet & check if we should run a backup
		projectFolder := config.Folder + database.Name + "/"
		shouldBackup, err := unitski.CheckProjectFolder(projectFolder, database.Interval)
		if err != nil {
			// TODO: Sentry
			log.Println(err.Error())
			continue
		} else if !shouldBackup.Any() {
			log.Println("No backup required today for: " + database.Name)
			continue
		}

		// Determine the dump file
		dumpToFile := projectFolder + database.Name + "_" + date + ".sql"

		// Execute the dump
		log.Println("Starting dump to file: " + dumpToFile)
		err = unitski.DumpMySqlDatabase(cli, ctx, database, dumpToFile)
		if err != nil {
			// TODO: Sentry log this
			log.Println("Failed to dump MySQL database of " + database.Name + ": " + err.Error())
			continue
		}

		// Compress the dump
		log.Println("Compressing file: " + dumpToFile)
		compressedFile, err := unitski.Compress(dumpToFile)
		if err != nil {
			// TODO Sentry
			log.Print("Failed to compress file: " + dumpToFile + " | Err: " + err.Error())
			continue
		}

		// Rotate the file through
		log.Println("Rotating result file into backups")
		err = unitski.RotateFile(compressedFile, shouldBackup, database.Interval)
		if err != nil {
			// TODO Sentry
			log.Print("Error while rotating file: " + err.Error())
			continue
		}

		// TODO: Queue sync

		// All done?
	}
}

func files(config unitski.BackupConfig) {
	date := time.Now().Format("2006-01-02")

	// Loop through each database
	for _, fileBackup := range config.Files {
		if !fileBackup.Enabled {
			log.Println("Skipping files backup: " + fileBackup.Name + " (is disabled)")
			continue
		}

		log.Println("Starting backup of files: " + fileBackup.Name)

		// Create the project folder if not done yet & check if we should run a backup
		projectFolder := config.Folder + fileBackup.Name + "/"
		shouldBackup, err := unitski.CheckProjectFolder(projectFolder, fileBackup.Interval)
		if err != nil {
			// TODO: Sentry
			log.Println(err.Error())
			continue
		} else if !shouldBackup.Any() {
			log.Println("No backup required today for: " + fileBackup.Name)
			continue
		}

		// Determine the target tar file
		tarBallFile := projectFolder + fileBackup.Name + "_" + date + ".tar"
		if fileBackup.Compress {
			tarBallFile = tarBallFile + ".gz"
		}

		// Create the tar ball
		log.Println("Creating tar ball: " + tarBallFile)
		err = unitski.CreateTarBall(tarBallFile, fileBackup.Files, fileBackup.Exclude)
		if err != nil {
			// TODO Sentry
			log.Print("Error while creating tar ball: " + err.Error())
			continue
		}

		// Rotate the file through
		log.Println("Rotating result file into backups")
		err = unitski.RotateFile(tarBallFile, shouldBackup, fileBackup.Interval)
		if err != nil {
			// TODO Sentry
			log.Print("Error while rotating file: " + err.Error())
			continue
		}

		// TODO: Queue sync

		// All done?
	}
}
