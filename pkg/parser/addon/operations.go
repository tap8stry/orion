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
	"errors"
	"fmt"
	"strings"

	"github.com/tap8stry/orion/pkg/common"
	"golang.org/x/crypto/sha3"
)

// processCurl parses curl command
func processCurl(args []string, workdir string, stageargs map[string]string) (string, common.Trace) {
	trace := common.Trace{
		Command:     curlOperation,
		Source:      "",
		Destination: workdir,
		Workdir:     workdir,
	}
	origin := ""
	for i, arg := range args {
		arg = replaceArgEnvVariable(arg, stageargs)
		if strings.HasPrefix(arg, "http") {
			trace.Source = arg
			origin = trace.Source //the installation source
		}
		if (strings.HasPrefix(arg, "--output") || strings.HasPrefix(arg, "-o") || strings.EqualFold(arg, ">")) && len(args) > i+1 {
			trace.Destination = replaceArgEnvVariable(args[i+1], stageargs) //update the destination
			if !strings.HasPrefix(trace.Destination, "/") {
				trace.Destination = strings.TrimSuffix(workdir, "/") + "/" + trace.Destination
			}
		}
	}
	return origin, trace
}

// processWget parses wget command
func processWget(args []string, workdir string, stageargs map[string]string) (string, common.Trace) {
	trace := common.Trace{
		Command:     wgetOperation,
		Source:      "",
		Destination: workdir,
		Workdir:     workdir,
	}
	origin := ""
	defaultName := ""
	useDefaultName := true

	for i, arg := range args {
		arg = common.TrimQuoteMarks(arg)
		arg = replaceArgEnvVariable(arg, stageargs)
		if strings.HasPrefix(arg, "http") {
			trace.Source = arg
			origin = trace.Source
			splits := strings.Split(arg, "/")
			defaultName = replaceArgEnvVariable(splits[len(splits)-1], stageargs)
		}
		if strings.HasPrefix(arg, "-O") && len(args) > i+1 { //filename to store download in workdir
			trace.Destination = replaceArgEnvVariable(args[i+1], stageargs)
			if !strings.HasPrefix(trace.Destination, "/") {
				trace.Destination = strings.TrimSuffix(workdir, "/") + "/" + trace.Destination
			}
			i++
			useDefaultName = false
		}
		if strings.HasPrefix(arg, "-P") && len(args) > i+1 { //dir where download will be stored
			trace.Destination = replaceArgEnvVariable(args[i+1], stageargs)
			if !strings.HasPrefix(trace.Destination, "/") {
				trace.Destination = strings.TrimSuffix(workdir, "/") + "/" + trace.Destination
			}
		}
	}
	if useDefaultName {
		trace.Destination = strings.TrimSuffix(trace.Destination, "/") + "/" + defaultName
	}
	return origin, trace
}

//processGitClone parses git clone command
func processGitClone(args []string, workdir string, stageargs map[string]string) (string, common.Trace) {
	//ssh://[user@]host.xz[:port]/path/to/repo.git/
	//http[s]://host.xz[:port]/path/to/repo.git/

	trace := common.Trace{
		Command:     gitCloneOperation,
		Source:      "",
		Destination: workdir,
		Workdir:     workdir,
	}
	origin := ""

	for i, arg := range args {
		if strings.HasPrefix(arg, "ssh://") || strings.HasPrefix(arg, "https://") || strings.HasPrefix(arg, "http://") {
			trace.Source = replaceArgEnvVariable(arg, stageargs)
			origin = trace.Source
			if i < len(args)-1 {
				trace.Destination = replaceArgEnvVariable(args[i+1], stageargs)
			} else {
				splits := strings.Split(trace.Source, "/")
				trace.Destination = splits[len(splits)-1]
			}
			if !strings.HasPrefix(trace.Destination, "/") {
				trace.Destination = strings.TrimSuffix(workdir, "/") + "/" + trace.Destination
			}
			return origin, trace
		}
		if strings.EqualFold(arg, "-b") && len(args) > i+1 { //add git branch
			trace.Command += " -b " + args[i+1]
		}
	}
	return origin, trace
}

//processGitCheckout parses git checkout command, recognizes only 'git checkout <branch or commit id>'
func processGitCheckout(args []string, workdir, giturl string, stageargs map[string]string) common.Trace {
	trace := common.Trace{
		Command:     gitCheckoutOperation,
		Source:      "",
		Destination: workdir,
		Workdir:     workdir,
	}
	if len(args) == 3 {
		trace.Source = replaceArgEnvVariable(args[2], stageargs)
		trace.Destination = replaceArgEnvVariable(workdir, stageargs)
	}
	return trace
}

// processTar parses tar command
func processTar(args []string, workdir string, stageargs map[string]string) common.Trace {
	trace := common.Trace{
		Command:     tarOperation,
		Source:      "",
		Destination: workdir,
		Workdir:     workdir,
	}
	for j := 1; j < len(args); j++ { //start from j=1 to skip "tar")
		if (args[j] == "-C" || strings.HasPrefix(args[j], "--directory")) && len(args) > j+1 {
			trace.Destination = replaceArgEnvVariable(args[j+1], stageargs)
			j++
			continue
		}
		if args[j] == "-xJC" && len(args) > j+1 {
			trace.Destination = replaceArgEnvVariable(args[j+1], stageargs)
			trace.Command += fmt.Sprintf(" %s", args[j])
			j++
			continue
		}
		if args[j] == "-f" && len(args) > j+1 {
			trace.Source = replaceArgEnvVariable(args[j+1], stageargs)
			j++
			continue
		}
		if args[j] == "-xfC" && len(args) > j+2 {
			trace.Destination = replaceArgEnvVariable(args[j+1], stageargs)
			trace.Source = replaceArgEnvVariable(args[j+2], stageargs)
			trace.Command += fmt.Sprintf(" %s", args[j])
			j += 2
			continue
		}
		if strings.Contains(args[j], "x") && strings.Contains(args[j], "f") && len(args) > j+1 { //could be -xvf, -xf, xvf, -zxvf...
			trace.Command += fmt.Sprintf(" %s", args[j])
			trace.Source = replaceArgEnvVariable(args[j+1], stageargs)
			j++
			continue
		}
	}
	if !strings.HasPrefix(trace.Source, "/") {
		trace.Source = strings.TrimSuffix(workdir, "/") + "/" + trace.Source
	}
	return trace
}

// processZip parses zip command
func processUnzip(args []string, workdir string, stageargs map[string]string) common.Trace {
	trace := common.Trace{
		Command:     unzipOperation,
		Source:      "",
		Destination: workdir,
		Workdir:     workdir,
	}

	/* 	unzip [-Z] [-opts[modifiers]] file[.zip] [list] [-x xlist] [-d exdir]
	   	unzip latest.zip
		unzip filename.zip -d /path/to/directory
		unzip filename.zip -x file1-to-exclude file2-to-exclude
		unzip -P PasswOrd filename.zip  */
	for j := 1; j < len(args); j++ { //skip "unzip", e.g unzip gradle-*.zip
		if strings.HasPrefix(args[j], "-d") && len(args) > j+1 {
			trace.Destination = replaceArgEnvVariable(args[j+1], stageargs)
			j++
			continue
		}
		if strings.HasPrefix(args[j], "-P") && len(args) > j+1 { //unzip -P <password> file.zip
			j++
			continue
		}
		if !strings.HasPrefix(args[j], "-") && len(trace.Source) == 0 {
			trace.Source = replaceArgEnvVariable(args[j], stageargs)
			if !strings.HasPrefix(trace.Source, "/") {
				trace.Source = strings.TrimSuffix(workdir, "/") + "/" + trace.Source
			}
			continue
		}
	}
	return trace
}

// processCp parses cp command
func processCp(args []string, workdir string, stageargs map[string]string) common.Trace {
	trace := common.Trace{
		Command: cpOperation,
		Workdir: workdir,
	}
	for j := 1; j < len(args); j++ { //skip j=0 ("cp")
		if !strings.HasPrefix(args[j], "-") && len(args) > j+1 {
			trace.Source = replaceArgEnvVariable(args[j], stageargs)
			if !strings.HasPrefix(trace.Source, "/") {
				trace.Source = strings.TrimSuffix(workdir, "/") + "/" + trace.Source
			}
			trace.Destination = replaceArgEnvVariable(args[j+1], stageargs)
			if !strings.HasPrefix(trace.Destination, "/") {
				trace.Destination = strings.TrimSuffix(workdir, "/") + "/" + trace.Destination
			}
			break
		}
	}
	return trace
}

// processCd parses change directory command and returns the current dir (not support 'cd ../')
func processCd(args []string, workdir string, stageargs map[string]string) string {
	//git clone https://github.com/alievk/avatarify-python.git /app/avatarify && cd /app/avatarify && git checkout <branch>
	var currentdir string
	if len(args) == 2 {
		if strings.EqualFold(args[1], "~") {
			currentdir = "~"
		} else {
			currentdir = replaceArgEnvVariable(args[1], stageargs)
			if !strings.HasPrefix(currentdir, "/") {
				currentdir = strings.TrimSuffix(workdir, "/") + "/" + currentdir
			}
		}
	}
	return currentdir
}

// processMv parses mv command
func processMv(args []string, workdir string, stageargs map[string]string) common.Trace {
	trace := common.Trace{
		Command: mvOperation,
		Workdir: workdir,
	}
	for j := 1; j < len(args); j++ { //skip j=0 ("mv")
		if !strings.HasPrefix(args[j], "-") && len(args) > j+1 {
			trace.Source = replaceArgEnvVariable(args[j], stageargs)
			if !strings.HasPrefix(trace.Source, "/") {
				trace.Source = strings.TrimSuffix(workdir, "/") + "/" + trace.Source
			}
			trace.Destination = replaceArgEnvVariable(args[j+1], stageargs)
			if !strings.HasPrefix(trace.Destination, "/") {
				trace.Destination = strings.TrimSuffix(workdir, "/") + "/" + trace.Destination
			}
			break
		}
	}
	return trace
}

// processCopyAdd parses COPY and ADD operation, e.g. COPY (or ADD) [--chown=<user>:<group> --from=<buildstage>] <src>... <dest>
func processCopyAdd(args []string, workdir, namespace string, stageargs map[string]string) (common.InstallTrace, error) {
	installTrace := common.InstallTrace{}
	trace := common.Trace{
		Command: args[0],
		Workdir: workdir,
	}
	m := make(map[int]common.Trace)
	for j := 1; j < len(args); j++ { //skip j=0 ("COPY")
		if !strings.HasPrefix(args[j], "--") && len(args) > j+1 {
			trace.Source = replaceArgEnvVariable(strings.Join(args[j:len(args)-1], ","), stageargs)
			if strings.HasPrefix(trace.Source, "http") {
				installTrace.Origin = replaceArgEnvVariable(args[j], stageargs)
			} else {
				installTrace.Origin = fmt.Sprintf("%s", namespace)
				installTrace.OriginHash = fmt.Sprintf("%x", sha3.Sum256([]byte(namespace)))
			}
			trace.Destination = replaceArgEnvVariable(args[len(args)-1], stageargs)
			if !strings.HasPrefix(trace.Destination, "/") {
				if strings.EqualFold(trace.Destination, "./") {
					trace.Destination = strings.TrimSuffix(workdir, "/") + "/."
				} else {
					trace.Destination = strings.TrimSuffix(workdir, "/") + "/" + strings.TrimSpace(trace.Destination)
				}
			}
			m[0] = trace
			installTrace.Traces = m
			return installTrace, nil
		}
		if strings.HasPrefix(args[j], "--from=") && len(args) > j+2 {
			for i := j + 1; i < len(args); i++ {
				if strings.HasPrefix(args[i], "--") {
					continue
				}
				trace.Source = replaceArgEnvVariable(strings.Join(args[i:len(args)-1], ","), stageargs)
				trace.Destination = replaceArgEnvVariable(args[len(args)-1], stageargs)
				if !strings.HasPrefix(trace.Destination, "/") {
					if strings.EqualFold(trace.Destination, "./") {
						trace.Destination = strings.TrimSuffix(workdir, "/") + "/."
					} else {
						trace.Destination = strings.TrimSuffix(workdir, "/") + "/" + trace.Destination
					}
				}
				m[0] = trace
				installTrace.Traces = m
				installTrace.Origin = fmt.Sprintf("buildstage:%s:%s", strings.TrimPrefix(args[j], "--from="), trace.Source)
				return installTrace, nil
			}
		}
	}
	return installTrace, errors.New("no trace can be produced")
}
