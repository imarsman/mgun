package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
)

var applicationRelativePath = "/cmd/mgun/"

var cYellow = color.New(color.FgYellow, color.Bold)
var cGreen = color.New(color.FgGreen, color.Bold)

// AppName get the application name used in project hierarchy
func AppName() string {
	return "mgun"
}

// GetFileContents get file contents. Useful for testing.
func GetFileContents(path string) (contents []byte, err error) {
	// path = TestBaseDir() + "/testfiles/" + path
	path, err = GetCleanPath(path)
	if err != nil {
		return []byte{}, fmt.Errorf("got error: %v", err)
	}
	content, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		return []byte{}, fmt.Errorf("could not read file: %v", readErr)
	}
	return content, nil
}

// AdjustPath validate path relative to install or testing base.
// If a path it absolute it will be validated as-is. If it is a relative path
// it will be validated relative to the application binary's parent for live
// and relative to the /bin/test/appname dir for testing.
func AdjustPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("empty path to test %s", p)
	}
	if filepath.IsAbs(p) == true {
		// Handle error for file not existing
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err != nil {
				msg := fmt.Sprintf("Path %s does not exist\n", p)
				cYellow.Print(msg)
				return "", err
			}
			// Print out extra info for testing
			if RunningTest() == true {
				msg := fmt.Sprintf("Path %s exists\n", p)
				cGreen.Print(msg)
			}
		}
		// No error so return path and nil
		return p, nil
	}
	if RunningTest() {
		p = filepath.FromSlash(filepath.Join(TestBaseDir(), AppName(), p))
		// fmt.Println("Filepath: ", p)
		// Handle error for file not existing
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err != nil {
				msg := fmt.Sprintf("Path %s does not exist\n", p)
				cYellow.Print(msg)
				return "", err
			}
			// Print out extra info for testing
			if RunningTest() == true {
				msg := fmt.Sprintf("Path %s exists\n", p)
				cGreen.Print(msg)
			}
		}
		// No error so return path and nil
		return p, nil
	}
	p = filepath.FromSlash(filepath.Join(ExeParentDir(), p))
	if _, err := os.Stat(p); os.IsNotExist(err) {
		// Handle error for file not existing
		if err != nil {
			msg := fmt.Sprintf("Path %s does not exist\n", p)
			cYellow.Print(msg)
			return "", err
		}
		// Print out extra info for testing
		if RunningTest() == true {
			msg := fmt.Sprintf("Path %s exists\n", p)
			cGreen.Print(msg)
		}
	}
	// No error so return path and nil
	return p, nil
}

// ProjectDir get root directory path for project
func ProjectDir() string {
	// Get base of code part of this executable
	dir := RootDir()
	// Go up two levels
	dir = path.Dir(path.Dir(dir))
	return filepath.Dir(dir)
}

// RootDir get root directory path for whatver is running.
// In the case of a test it is based on main.
// In the case of a compiled executable it is the dir for the executable.
func RootDir() string {
	// Get path to the base for code
	_, b, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(b))
	// fmt.Printf("dir: %s", dir)
	return filepath.Dir(dir)
}

// GetCleanPath get a cleaned up and verified path
func GetCleanPath(path string) (string, error) {
	path = filepath.Clean(path)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return path, fmt.Errorf("Path %s does not exist", path)
	}
	return filepath.FromSlash(path), nil
}

// TestBaseDir base directory for testing
func TestBaseDir() string {
	// Get base of code part of this executable
	// fmt.Printf("project dir: %s\n", ProjectDir())
	dir := path.Join(ProjectDir(), "/bin/test/")
	// fmt.Printf("dir: %s", dir)
	return dir
}

// ExeParentDir the parent directory for the executable
func ExeParentDir() string {
	dir := ""
	if !RunningTest() {
		// Get path to the executable
		// The executable is in the path
		exePath, err := os.Executable()
		if err != nil {
			panic("Could not get current directorty")
		}
		// fmt.Println(exePath)
		// Get path without executable in path
		baseDir := filepath.Dir(exePath)
		// Get parent of the executable dir
		dir = filepath.Dir(baseDir)
	}
	return dir
}

// RunningTest checks whether code is running in a test
func RunningTest() bool {
	return strings.HasSuffix(os.Args[0], ".test")
}

// CheckPath check a path from config
func CheckPath(path string) error {
	if RunningTest() {
		context := "test"

		if path == "" {
			return fmt.Errorf("No directory specified. Context: %s", context)
		}
		if strings.HasPrefix(path, "/") {
			testPath, err := filepath.Abs(ExeParentDir() + "/" + path)
			if err != nil {
				return fmt.Errorf(fmt.Sprintf("Problem obtaining path: %s. Error: %s. Context: %s", testPath, err, context))
			}
			if _, err := os.Stat(testPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("Problem finding directory path: %s. Context: %s", testPath, context)
				}
			}
		} else {
			testDir := filepath.Dir(RootDir()) + "/test/" + path
			testPath, err := filepath.Abs(testDir)
			if err != nil {
				return fmt.Errorf("Problem obtaining path: %s. Context: %s", err, context)
			}
			if _, err := os.Stat(testPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("Problem finding directory path %s. Context: %s", testPath, context)
				}
			}
		}

	} else {
		context := "live"

		if path == "" {
			return fmt.Errorf("No path specified. Context: %s", context)
		}
		if strings.HasPrefix(path, "/") {
			testPath, err := filepath.Abs(ExeParentDir() + "/" + path)
			if err != nil {
				return fmt.Errorf(fmt.Sprintf("Problem obtaining path: %s. Error: %s. Context: %s", testPath, err, context))
			}
			if _, err := os.Stat(testPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("Problem finding directory path: %s. Context: %s", testPath, context)
				}
			}
		} else {
			testDir := ExeParentDir() + "/" + path
			testPath, err := filepath.Abs(testDir)
			if err != nil {
				return fmt.Errorf("Problem obtaining path: %s. Context: %s", err, context)
			}
			if _, err := os.Stat(testPath); err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("Problem finding directory path %s. Context: %s", testPath, context)
				}
			}
		}
	}
	return nil
}
