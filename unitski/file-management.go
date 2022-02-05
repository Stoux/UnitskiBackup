package unitski

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

const monthlyDir = "monthly/"
const weeklyDir = "weekly/"
const dailyDir = "daily/"
const fileDatePattern = "_(\\d{4}-\\d{2}-\\d{2})\\."

type FileError struct {
	msg string
}

func (error *FileError) Error() string {
	return error.msg
}

type FolderCreator struct {
	root string
	err  error
}

func (fc *FolderCreator) checkOrCreate(subFolder string, name string) {
	folder := fc.root + subFolder

	// Only execute if the previous call(s) didn't fail
	if fc.err != nil {
		return
	}

	// Create the project folder if not done yet
	if stat, dirErr := os.Stat(folder); os.IsNotExist(dirErr) {
		log.Println("Creating " + name + ": " + folder)
		if mkDirErr := os.Mkdir(folder, os.ModePerm); mkDirErr != nil {
			fc.err = &FileError{"Failed to create " + name + ": " + folder + " | " + mkDirErr.Error()}
		}
	} else if !stat.IsDir() {
		fc.err = &FileError{"'" + name + "' isn't a folder: " + folder}
	}
}

func (fc *FolderCreator) checkIfExists(subFolder string, file string) (bool, error) {
	file = fc.root + subFolder + file
	if _, err := os.Stat(file); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func (fc *FolderCreator) checkShouldBackup(
	subFolder string,
	file string,
	interval int,
	shouldBackupToday func() bool,
) (backup bool) {
	// Don't backup when there's no interval
	if fc.err != nil || interval <= 0 {
		return false
	}

	var err error

	// Check if the file already exists
	if exists, err := fc.checkIfExists(subFolder, file); err == nil && !exists {
		// Doesn't exist yet, check if we should make a back-up (today)
		if backup = shouldBackupToday(); !backup {
			// We shouldn't back-up today... but we might still want to if there are no back-ups yet?
			if previousBackups, err := getPreviousBackups(fc.root + subFolder); err == nil {
				backup = len(previousBackups) == 0
			}
		}
	} else if exists {
		log.Println("File " + subFolder + file + " already exists")
	}

	if err != nil {
		fc.err = err
	}

	return backup
}

type ShouldBackup struct {
	daily   bool
	weekly  bool
	monthly bool
}

func (sb *ShouldBackup) Any() bool {
	return sb.daily || sb.weekly || sb.monthly
}

// CheckProjectFolder checks whether the project folder is correctly backed-up & whether a backup should take place.
func CheckProjectFolder(projectFolder string, filename string, interval BackupInterval) (shouldBackup ShouldBackup, err error) {
	shouldBackup = ShouldBackup{}

	// Create the project folder structure if not done yet
	creator := FolderCreator{root: projectFolder}
	creator.checkOrCreate("", "root backup folder")
	creator.checkOrCreate(monthlyDir, "monthly backup folder")
	creator.checkOrCreate(weeklyDir, "weekly backup folder")
	creator.checkOrCreate(dailyDir, "daily backup folder")
	if creator.err != nil {
		return shouldBackup, creator.err
	}

	// Check that at least one interval is active
	if interval.Daily <= 0 && interval.Weekly <= 0 && interval.Monthly <= 0 {
		return shouldBackup, &FileError{"All intervals have 0, this backup will never run!"}
	}

	// Check whether backups should be made
	shouldBackup.daily = creator.checkShouldBackup(dailyDir, filename, interval.Daily, func() bool {
		return true // Every day
	})
	shouldBackup.weekly = creator.checkShouldBackup(weeklyDir, filename, interval.Weekly, func() bool {
		return time.Now().Weekday() == time.Monday // Every monday
	})
	shouldBackup.monthly = creator.checkShouldBackup(monthlyDir, filename, interval.Monthly, func() bool {
		return time.Now().Day() == 1 // First of the month
	})

	return shouldBackup, creator.err
}

func getPreviousBackups(folder string) (hasBackup []string, err error) {
	var result []string

	entries, err := os.ReadDir(folder)
	if err != nil {
		return result, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			if match, _ := regexp.MatchString(fileDatePattern, entry.Name()); match {
				result = append(result, entry.Name())
			}
		}
	}

	return result, nil
}

type BackupFolder struct {
	folder            string
	referencingFolder *BackupFolder // The child (faster change rate) that might reference this folder
}

// getChildSymlinkFor resolves whether this file is used by this folder or any of its possible referencing folders.
// Returns an absolute path to the symlink
// Might return an error of the file isn't a symlink
func (f *BackupFolder) getSymlinkFor(file string) (symlinkPath string, err error) {
	symlinkPath = f.folder + file

	// Check if the file exists in this folder
	if stat, err := os.Lstat(symlinkPath); err == nil {
		// We have the file, it should be a symlink
		if stat.Mode()&os.ModeSymlink == os.ModeSymlink {
			return symlinkPath, err
		} else {
			return "", &FileError{"Child folder has a file which should be a symlink, ready for rotate, but isn't: " + symlinkPath}
		}
	} else if !os.IsNotExist(err) {
		// Great if it doesn't exist, if there' something else wrong we got a problem.
		return "", err
	}

	// Check a possible referencing folder
	if f.referencingFolder != nil {
		return f.referencingFolder.getSymlinkFor(file)
	}

	// Nothing found
	return "", err
}

type FileRotator struct {
	rootFolder          string
	filename            string
	originalFilePath    string
	lowestLevelLocation string
	err                 error
}

func (r *FileRotator) backupTo(subFolder string, should bool) {
	// Check if we even should back up for this folder
	if r.err != nil || !should {
		return
	}

	toPath := r.rootFolder + subFolder + r.filename

	// We should back up to the given folder, check if we're the first to back up the file
	if r.lowestLevelLocation == "" {
		// We're the first, move the file to the folder
		if err := os.Rename(r.originalFilePath, toPath); err != nil {
			r.err = err
			return
		}
	} else {
		// The file is already moved, symlink to it
		// Relative path = 'Current Subfolder' => 'Go up, back to to rootFolder
		relativePath := "./../" + r.lowestLevelLocation + r.filename
		if err := os.Symlink(relativePath, toPath); err != nil {
			r.err = err
			return
		}
	}

	// Update the lowestLevelLocation to us so that lower levels go through our symlink
	// We need to do this as files might rotate from our parent to us, all lower level symlinks keep working
	r.lowestLevelLocation = subFolder
}

func (r *FileRotator) purge(backupType BackupFolder, keep int) {
	if r.err != nil || keep == 0 {
		return
	}

	// Retrieve all (previous) backups from the folder
	backups, err := getPreviousBackups(backupType.folder)
	if err != nil {
		r.err = err
		return
	}

	// Check if there are too many
	totalBackupsToBeDeleted := len(backups) - keep
	if totalBackupsToBeDeleted <= 0 {
		// Nothing to rotate
		return
	}

	// Sort them by oldest -> newest
	regex := regexp.MustCompile(fileDatePattern)
	toUnix := func(file string) int64 {
		date := regex.FindStringSubmatch(file)[0]
		if result, err := time.Parse("2006-01-02", date); err != nil {
			return result.Unix()
		} else {
			// Shouldn't happen as it got matched by previous function
			panic(err)
		}
	}
	sort.SliceStable(backups, func(i, j int) bool {
		return toUnix(backups[i]) < toUnix(backups[j])
	})

	// Slice the items that we need to delete
	toDelete := backups[:totalBackupsToBeDeleted]
	for _, deleteFile := range toDelete {
		deleteFileAbsPath := backupType.folder + deleteFile

		// Check if we should move the file to a referencing folder
		if backupType.referencingFolder != nil {
			if deletedFileDestination, err := backupType.referencingFolder.getSymlinkFor(deleteFile); err != nil {
				// Error occurred while resolving the symlink
				r.err = err
				return
			} else if deletedFileDestination != "" {
				// Found a symlink, remove that symlink
				if err := os.Remove(deletedFileDestination); err != nil {
					r.err = err
					return
				}

				// Move our file to the destination of the symlink
				// Note that our file might also be a symlink, this should be fine as it's a relative symlink with the same level of depth
				if err := os.Rename(deleteFileAbsPath, deletedFileDestination); err != nil {
					r.err = err
					return
				}

				// All good!
				log.Println("Moved " + deleteFileAbsPath + " to " + deletedFileDestination)
				continue
			}
		}

		// Nothing references this file. Just remove it.
		if err := os.Remove(deleteFileAbsPath); err != nil {
			r.err = err
			return
		}
	}
}

// RotateFile will rotate the given file into the backup folder
// This also removes any old files that are due for deletion
func RotateFile(createdFilePath string, shouldBackup ShouldBackup, interval BackupInterval) error {
	// Explode the filepath & store in a FileRotator
	rotator := FileRotator{
		rootFolder:       filepath.Dir(createdFilePath) + "/",
		filename:         filepath.Base(createdFilePath),
		originalFilePath: createdFilePath,
	}

	// Move the file to the 'oldest' folder
	rotator.backupTo(monthlyDir, shouldBackup.monthly)
	rotator.backupTo(weeklyDir, shouldBackup.weekly)
	rotator.backupTo(dailyDir, shouldBackup.daily)
	if rotator.err != nil {
		return rotator.err
	}

	// Create structure of folders
	dailyFolder := BackupFolder{folder: rotator.rootFolder + dailyDir}
	weeklyFolder := BackupFolder{rotator.rootFolder + weeklyDir, &dailyFolder}
	monthlyFolder := BackupFolder{rotator.rootFolder + monthlyDir, &weeklyFolder}

	// Rotate out any old files
	rotator.purge(dailyFolder, interval.Daily)
	rotator.purge(weeklyFolder, interval.Weekly)
	rotator.purge(monthlyFolder, interval.Monthly)
	if rotator.err != nil {
		return rotator.err
	}

	return nil
}
