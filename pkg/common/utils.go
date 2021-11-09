//
// Copyright 2020 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
			fmt.Printf("\nerror writing data to %q", filename)
			return
		}
		fmt.Printf("\nresults saved to: %q", filename)
	}
}
