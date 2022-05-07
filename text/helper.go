package text

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	gogetter "github.com/hashicorp/go-getter"
)

// Constants for helper file
const (
	formatSeperator   = ":"
	file              = "file"
	gogetterdirectory = "/go-getter-files/"
)

// Setting Supported Detectors
var Detectors = []gogetter.Detector{
	new(gogetter.GitHubDetector),
	new(gogetter.GitDetector),
	new(gogetter.BitBucketDetector),
	new(gogetter.S3Detector),
	new(gogetter.GCSDetector),
	new(gogetter.FileDetector),
}

func SafeRead(r io.Reader) string {
	data, _ := ioutil.ReadAll(r)
	return string(data)
}

func ResolveFile(filepath string) (string, error) {
	pwd, _ := os.Getwd()

	source, err := gogetter.Detect(filepath, "", Detectors)

	// If Not Valid Go Getter => return string content
	if source == "" {
		return filepath, nil
	}

	format := strings.Split(source, formatSeperator)

	// Checking If it is a file or not
	if format[0] == file {

		mydir, err := os.Getwd()
		// Error While geting pwd
		if err != nil {
			return "", err
		}

		fullFilePath := fmt.Sprintf(mydir+"%s", filepath)

		fileBytes, err := ioutil.ReadFile(fullFilePath)
		if err != nil {
			return "", err
		}
		return string(fileBytes), nil

	}

	// If it is not a file format Go Get It
	fullDestination := fmt.Sprintf(pwd+"%s%s", gogetterdirectory, filepath)
	err = gogetter.GetFile(fullDestination, filepath, gogetter.WithContext(context.Background()))

	// if err is nil => return the downloaded file path
	if err == nil {
		return fullDestination, nil
	}
	return source, err
}

func ResolveFiles(filespath string) (map[string]string, error) {
	results := map[string]string{}
	source, _ := gogetter.Detect(filespath, "", Detectors)

	// If Not Valid Go Getter => return string content
	if source == "" {
		return results, nil
	}

	//check if source is files
	pwd, _ := os.Getwd()
	format := strings.Split(source, formatSeperator)
	if format[0] == file {
		// walking thorugh the directory to read the files
		err := filepath.Walk(pwd+filespath, func(path string, info os.FileInfo, err error) error {

			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			// removing pwd from the full path
			pathRemovedPwd := strings.Split(path, pwd)
			// setting the result in the pwd
			results[pathRemovedPwd[1]], err = ResolveFile(pathRemovedPwd[1])
			if err != nil {
				return err
			}
			return nil

		})

		return results, err

	}

	// Get The Dir then
	fullDestination := fmt.Sprintf(pwd+"%s%s", gogetterdirectory, filespath)
	err := gogetter.Get(fullDestination, filespath, gogetter.WithContext(context.Background()))
	fmt.Println(err)
	if err != nil {
		results[filespath] = filespath
	} else {
		// set files path as repo link
		results[filespath] = fullDestination
	}
	return results, nil
}
