package unitski

import (
	"log"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

func matchFirstRegexGroupToInt(regex *regexp.Regexp, text string) (int64, error) {
	// Should match '[size] [name of file]'
	match := regex.FindStringSubmatch(text)
	if match == nil {
		return 0, &FileError{"du output did not match expected format: " + text}
	}

	// Convert to a i64
	if result, err := strconv.ParseInt(match[1], 10, 64); err != nil {
		return 0, err
	} else {
		return result, nil
	}
}

// GetDiskSpaceAvailable uses `df` to resolve the disk where the given folder is on and returns the number of kilobytes available on that disk.
func GetDiskSpaceAvailable(folder string) (result int64, err error) {
	// Always require at least 5GB on the disk (should be configurable?)
	defer func() {
		result -= 5_000_000
	}()

	// Run the command
	output, err := exec.Command("df", "-k", folder).Output()
	if err != nil {
		return 0, err
	}

	// Split lines, first one should be header, second the disk
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, &FileError{"`df` output doesn't match expected output: " + string(output)}
	}

	// Match the file line to fetch the available value
	// Format: [Filesystem] [Size] [Used] [Available] [...]
	return matchFirstRegexGroupToInt(
		regexp.MustCompile("^(?:[^\\s]+\\s+){3}(\\d+)\\s+"),
		lines[1],
	)
}

// GetFolderSize checks the total size in kilobytes of the given folder using `du`
// Do note that exclude is not supported on Mac OS X platforms
func GetFolderSize(folder string, exclude []string) (int64, error) {
	// Build the arguments for the command
	arguments := []string{"-s", "-k", folder}
	if exclude != nil && runtime.GOOS != "darwin" {
		// Darwin = Mac
		for _, pattern := range exclude {
			arguments = append(arguments, "--exclude", pattern)
		}
	}

	// Run the command
	output, err := exec.Command("du", arguments...).Output()
	if err != nil {
		return 0, err
	}

	return matchFirstRegexGroupToInt(
		regexp.MustCompile("^(\\d+)\\s.+"),
		string(output),
	)
}

// Compress the given file using gzip @ max compression rating
// Returns the name of the compressed file.
func Compress(file string) (compressedFile string, err error) {
	p := exec.Command(
		"gzip",
		"-9",
		file,
	)
	if err := p.Run(); err != nil {
		return "", err
	}
	return file + ".gz", nil
}

// CreateTarBall creates a (possibly compressed) tar ball of the given files with the given exclude patterns.
// It will try to check if enough disk space is available if supported by the OS. Do note that this doesn't take any compression into account.
// (Mac OS X doesn't support --exclude for du commands)
func CreateTarBall(targetFilePath string, files []string, exclude []string) error {
	// Fetch the available space we have on the target disk / folder
	if availableSpace, err := GetDiskSpaceAvailable(filepath.Dir(targetFilePath)); err != nil {
		return err
	} else {
		// Get the total size of required size
		var requiredSpace int64
		for _, targetFile := range files {
			if fileSize, err := GetFolderSize(targetFile, exclude); err == nil {
				requiredSpace += fileSize
			} else {
				return err
			}
		}

		// Check if we have enough space
		if requiredSpace > availableSpace {
			return &FileError{"Not enough disk space available"}
		}
	}

	// Build the argument list for the tar command
	// TODO: Ability to set compression level?
	var tarArguments []string

	// => Add all excluded patterns
	for _, excludePattern := range exclude {
		tarArguments = append(tarArguments, "--exclude", excludePattern)
	}

	// => Main arguments for building the archive
	tarArguments = append(tarArguments,
		"-c",           // Create a new archive
		"-f",           // With the file name
		targetFilePath, // This file
	)

	// => Add all files that should be added to the archive
	tarArguments = append(tarArguments, files...)

	// Create the tar ball
	if output, err := exec.Command("tar", tarArguments...).CombinedOutput(); err != nil {
		log.Println(output)
		return err
	}

	return nil
}
