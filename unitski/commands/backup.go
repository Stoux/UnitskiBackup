package commands

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/getsentry/sentry-go"
	"log"
	"path/filepath"
	"time"
	"unitski-backup/unitski"
)

// Sync will trigger a full sync of all databases & files in the given config file.
func Sync(configFilePath string) {
	fmt.Println("Running...")
	unitski.SetLogger()
	log.Println("---- Starting backup routine")

	// Load config
	config := unitski.LoadConfig(configFilePath)

	// Init docker
	cli, ctx := unitski.InitDocker()

	// Backup DBs
	databases(cli, ctx, config)
	// Backup files
	files(config)

	// TODO: Async
	// Check if required commands are available
	// Start 2 threads: 1 for processing backups, 1 for syncing to backup

	log.Println("---- All done!")
	fmt.Println("Done.")
}

func databases(cli *client.Client, ctx context.Context, config unitski.BackupConfig) {
	date := time.Now().Format("2006-01-02")

	// Loop through each database
	for _, database := range config.Databases {
		if !database.Enabled {
			log.Println("[info] Skipping backup of database: " + database.Name + " (is disabled)")
			continue
		}

		log.Println("[info] Starting backup of database: " + database.Name)

		// Determine the dump file
		projectFolder := config.Folder + database.Name + "/"
		dumpToFile := projectFolder + database.Name + "_" + date + ".sql"

		// Create the project folder if not done yet & check if we should run a backup
		shouldBackup, err := unitski.CheckProjectFolder(projectFolder, filepath.Base(dumpToFile+".gz"), database.Interval)
		if err != nil {
			log.Println("[error] ", err.Error())
			sentry.CaptureException(err)
			continue
		} else if !shouldBackup.Any() {
			log.Println("[info] No backup required today for: " + database.Name)
			continue
		}

		// Execute the dump
		log.Println("[info] Starting dump to file: " + dumpToFile)
		err = unitski.DumpMySqlDatabase(cli, ctx, database, dumpToFile)
		if err != nil {
			log.Println("[error] Failed to dump MySQL database of " + database.Name + ": " + err.Error())
			sentry.CaptureException(err)
			continue
		}

		// Compress the dump
		log.Println("[info] Compressing file: " + dumpToFile)
		compressedFile, err := unitski.Compress(dumpToFile)
		if err != nil {
			log.Print("[error] Failed to compress file: " + dumpToFile + " | Err: " + err.Error())
			sentry.CaptureException(err)
			continue
		}

		// Rotate the file through
		log.Println("Rotating result file into backups")
		err = unitski.RotateFile(compressedFile, shouldBackup, database.Interval)
		if err != nil {
			log.Print("[error] Error while rotating file: " + err.Error())
			sentry.CaptureException(err)
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
			log.Println("[info] Skipping files backup: " + fileBackup.Name + " (is disabled)")
			continue
		}

		log.Println("[info] Starting backup of files: " + fileBackup.Name)

		// Determine the target tar file
		projectFolder := config.Folder + fileBackup.Name + "/"
		tarBallFile := projectFolder + fileBackup.Name + "_" + date + ".tar"
		if fileBackup.Compress {
			tarBallFile = tarBallFile + ".gz"
		}

		// Create the project folder if not done yet & check if we should run a backup
		shouldBackup, err := unitski.CheckProjectFolder(projectFolder, filepath.Base(tarBallFile), fileBackup.Interval)
		if err != nil {
			log.Println("[error] ", err.Error())
			sentry.CaptureException(err)
			continue
		} else if !shouldBackup.Any() {
			log.Println("[info] No backup required today for: " + fileBackup.Name)
			continue
		}

		// Create the tar ball
		log.Println("[info] Creating tar ball: " + tarBallFile)
		err = unitski.CreateTarBall(tarBallFile, fileBackup.Files, fileBackup.Exclude)
		if err != nil {
			log.Print("[error] Error while creating tar ball: " + err.Error())
			sentry.CaptureException(err)
			continue
		}

		// Rotate the file through
		log.Println("[info] Rotating result file into backups")
		err = unitski.RotateFile(tarBallFile, shouldBackup, fileBackup.Interval)
		if err != nil {
			log.Print("[error] Error while rotating file: " + err.Error())
			sentry.CaptureException(err)
			continue
		}

		// TODO: Queue sync

		// All done?
	}
}
