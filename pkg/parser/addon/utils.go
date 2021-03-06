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

package addon

import (
	"fmt"
	"strings"

	"github.com/tap8stry/orion/pkg/common"
)

//RUN wget http://nodejs.org/dist/v14.17.6/node-v14.17.6-linux-$ARCH.tar.gz -O /tmp/node.tar.gz   && tar -C /usr/local --strip-components 1 -xzf /tmp/node.tar.gz &&   rm -rf /home/jhipster/.cache/ /var/lib/apt/lists/*  /tmp/* /var/tmp/*

var osPkgMgmtTools = []string{
	"apk  ",
	"apt ",
	"apt-get",
	"dpkg ",
	"yum ",
	"rpm ",
	"deb ",
}

// replaceArgEnvVariable repleces the variable with its value and removes quotation marks if exist
func replaceArgEnvVariable(str string, args map[string]string) string {
	newStr := str
	for key, value := range args {
		key1 := fmt.Sprintf("$%s", key)
		key2 := fmt.Sprintf("${%s}", key)
		key3 := "${" + key + "%%[a-z]*}"                      //used in python wget url, e.g. ${PYTHON_VERSION%%[a-z]*}
		if strings.Contains(newStr, key1) && len(value) > 0 { //replace only if the value is not empty
			newStr = strings.ReplaceAll(newStr, key1, value)
		}
		if strings.Contains(newStr, key2) && len(value) > 0 {
			newStr = strings.ReplaceAll(newStr, key2, value)
		}
		if strings.Contains(newStr, key3) && len(value) > 0 {
			newStr = strings.ReplaceAll(newStr, key3, value)
		}
	}
	newStr = common.TrimQuoteMarks(newStr)
	return newStr
}

// isOsInstall checks if the cmd is about OS package install
func isOsInstall(cmd string) bool {
	for _, osPkgCmd := range osPkgMgmtTools {
		if strings.Contains(cmd, osPkgCmd) {
			return true
		}
	}
	return false
}

// existInInstallTrace checks if the source is in a destination of previous steps, therefore belongs to the same curl/wget installation
func existInInstallTrace(traces map[int]common.Trace, source string) bool {
	for i := 0; i < len(traces); i++ {
		if strings.Contains(source, "*") { // source = "gradle-*.zip"
			if strings.Contains(traces[i].Destination, source[:strings.Index(source, "*")]) {
				return true
			}
		} else {
			if strings.HasPrefix(source, traces[i].Destination) {
				return true
			}
		}
	}
	return false
}

// checkEarlierInstalls checks if the source of trace matches a destination of previous steps of an install, add adds to the install's traces
func checkEarlierInstalls(installs *[]common.InstallTrace, trace common.Trace) {

	for _, install := range *installs {
		for i := 0; i < len(install.Traces); i++ {
			if strings.Contains(trace.Source, "*") { // source = "gradle-*.zip"
				if strings.Contains(install.Traces[i].Destination, trace.Source[:strings.Index(trace.Source, "*")]) {
					m := make(map[int]common.Trace)
					m = install.Traces
					m[len(install.Traces)] = trace
					install.Traces = m
					return
				}
			} else {
				if strings.HasPrefix(trace.Source, install.Traces[i].Destination) {
					m := make(map[int]common.Trace)
					m = install.Traces
					m[len(install.Traces)] = trace
					install.Traces = m
					return
				}
			}
		}
	}
	return
}
