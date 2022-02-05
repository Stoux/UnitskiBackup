# Unitski Backup

A little GoLang program that back-ups docker MySQL(-esque) containers & just plain file paths on my servers. Highly
tailored to my servers so high chance this is not what you're looking for and slim-chance I'm gonna support your server,
but who knows. You can always try and ask.

## Features

- Ability to dump, tar & compress the database from Docker MySQL/MariaDB containers
- Ability to dump, tar & optionally compress files on a server
- Automatic backup file rotation with the ability to specify how many backups should be kept (daily, weekly, monthly)

## Instructions

### Server / script pre-requisites:

- Runs as root (or access to all files & docker socket without authorization)
- A linux distro or OS X (mediocre support)
- Docker instance / socket is on the current server, default settings

### Setup

- Create a folder for the script, i.e. `/opt/backup-management/`
    - Make sure only root has access (`chown root:root`, `chown 600`)
    - Make sure
      to [set the default file bits for the folder](https://unix.stackexchange.com/questions/1314/how-to-set-default-file-permissions-for-all-folders-files-in-a-directory)
- Create the backup folder, i.e. `/opt/backup-management/backups/`
- Create a config file based on the [sample.json](sample.json)
- Run nightly cronjob: `unitski-backup backup path-to-config.json`

### Build from source

Run `GOOS=[platform] GOARCH=[arch] go build -v ./`.

Example for Linux distro /w AMD based cpu: `GOOS=linux GOARCH=amd64 go build -v ./`

## TODOs

- Sync files using rsync to different mount (and maybe later different server)
- Use routines to run multiple dumps in parallel
- Ability to set compression level through the config
- Ability to add a new database/file backup through the CLI
- Ability to test the configuration file through the CLI
- Sentry error reporting