package unitski

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
)

type BackupConfig struct {
	Folder     string                 `json:"folder"`
	SyncFolder string                 `json:"sync-folder"`
	Databases  []BackupConfigDatabase `json:"databases"`
	Files      []BackupConfigFiles    `json:"files"`
}

type BackupConfigDatabase struct {
	Name      string         `json:"name"`
	Enabled   bool           `json:"enabled"`
	Interval  BackupInterval `json:"interval"`
	Container string         `json:"container"`
	User      BackupVariable `json:"user"`
	Password  BackupVariable `json:"password"`
	Database  BackupVariable `json:"database"`
}

type BackupConfigFiles struct {
	Name                       string         `json:"name"`
	Enabled                    bool           `json:"enabled"`
	Interval                   BackupInterval `json:"interval"`
	Files                      []string       `json:"files"`
	Exclude                    []string       `json:"exclude"`
	Compress                   bool           `json:"compress"`
	RotateSyncedMonthlyBackups bool           `json:"rotate-synced-monthly-backups"`
}

type BackupInterval struct {
	Daily   int `json:"daily"`
	Weekly  int `json:"weekly"`
	Monthly int `json:"monthly"`
}

type BackupVariableType string

const (
	VarTypeConstant  BackupVariableType = "constant"
	VarTypeDockerEnv BackupVariableType = "env"
)

type BackupVariable struct {
	VarType BackupVariableType `json:"type"`
	Value   string             `json:"value"`
}

func LoadConfig(path string) BackupConfig {
	config := loadFromFile(path)
	validate(config)

	return config
}

func loadFromFile(path string) BackupConfig {
	jsonFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	var config BackupConfig
	byteValue, err := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		panic(err)
	}

	return config
}

func validate(config BackupConfig) {
	// Build a regex that makes sure the project names are path-safe
	allowedNameFormat, regErr := regexp.Compile("^[a-z0-9\\-_]+$")
	if regErr != nil {
		panic(regErr)
	}

	// Build a map of all known names
	knownNames := map[string]bool{}

	// Check if all names are unique & are folder/path-safe
	// => Database
	for _, database := range config.Databases {
		checkName(allowedNameFormat, knownNames, database.Name)
		knownNames[database.Name] = true
	}

	// => Other
	for _, fileBackup := range config.Files {
		checkName(allowedNameFormat, knownNames, fileBackup.Name)
		knownNames[fileBackup.Name] = true
	}

	// Check if the target folder exists, is writable, is an absolute path & has trailing /
	folder := config.Folder
	if matched, _ := regexp.MatchString("^/.+/$", folder); !matched {
		panic("Folder should be an absolute path with trailing slash, this isn't: " + folder)
	}
	if stat, dirErr := os.Stat(folder); os.IsNotExist(dirErr) {
		panic("The backup folder doesn't exist: " + folder)
	} else if !stat.IsDir() {
		panic("Backup 'folder' isn't a folder: " + folder)
	}
}

func checkName(allowedNameFormat *regexp.Regexp, knownNames map[string]bool, name string) {
	// => Unique
	if _, ok := knownNames[name]; ok {
		panic("Found duplicate project name entry: " + name + " | All names need to be unique!")
	}

	// => Path-safe
	if !allowedNameFormat.MatchString(name) {
		panic("A project name needs to be lowercase & path-safe (a-z0-9-_), this isn't: " + name)
	}
}
