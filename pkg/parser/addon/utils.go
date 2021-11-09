//  ###############################################################################
//  # Licensed Materials - Property of IBM
//  # (c) Copyright IBM Corporation 2021. All Rights Reserved.
//  #
//  # Note to U.S. Government Users Restricted Rights:
//  # Use, duplication or disclosure restricted by GSA ADP Schedule
//  # Contract with IBM Corp.
//  ###############################################################################

package addon

import (
	"fmt"
	"strings"

	"github.com/tapestry-discover/pkg/common"
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
		if strings.Contains(newStr, key1) && len(value) > 0 { //replace only if the value is not empty
			newStr = strings.ReplaceAll(newStr, key1, value)
		}
		if strings.Contains(newStr, key2) && len(value) > 0 {
			newStr = strings.ReplaceAll(newStr, key2, value)
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
		if strings.HasPrefix(source, traces[i].Destination) {
			return true
		}
	}
	return false
}
