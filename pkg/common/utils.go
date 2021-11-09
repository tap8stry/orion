//  ###############################################################################
//  # Licensed Materials - Property of IBM
//  # (c) Copyright IBM Corporation 2021. All Rights Reserved.
//  #
//  # Note to U.S. Government Users Restricted Rights:
//  # Use, duplication or disclosure restricted by GSA ADP Schedule
//  # Contract with IBM Corp.
//  ###############################################################################

package common

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/logs"
)

// SearchFiles returns matched file patterns or nil if none found
func SearchFiles(repoDir, pattern string) []string {
	matchedFiles := []string{}
	filepath.Walk(repoDir, func(path string, f os.FileInfo, _ error) error {
		if f != nil && !f.IsDir() {
			r, err := regexp.MatchString(pattern, f.Name())
			if err == nil && r {
				matchedFiles = append(matchedFiles, path)
			}
		}
		return nil
	})
	return matchedFiles
}

// TrimQuoteMarks returns a string with its quotation marks removed
func TrimQuoteMarks(value string) string {
	str := value
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		str = value[1 : len(value)-1]
	}
	return str
}

func SaveFile(filename string, data []byte) {
	if len(data) > 0 {
		err := ioutil.WriteFile(filename, data, 0644)
		if err != nil {
			logs.Warn.Printf("error writing data to %q", filename)
			return
		}
		logs.Progress.Printf("results saved to: %q", filename)
	}
}
