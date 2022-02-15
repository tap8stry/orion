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

package icrcredhelper

import (
	"fmt"
	"os/exec"
)

func GetToken(apikey string) string {
	//login to ibmcloud
	cmd := exec.Command("ibmcloud", "login", "-a", "https://cloud.ibm.com", "--apikey", apikey, "-r", "us-south")
	if err := cmd.Run(); err != nil {
		fmt.Printf("\nerror in executing ibmcloud login: %s", err.Error())
		return ""
	}
	//get iam token
	dat, err := exec.Command("ibmcloud", "iam", "oauth-tokens").Output()
	if err != nil {
		fmt.Printf("\nerror in executing ibmcloud iam oauth-tokens: %s", err.Error())
		return ""
	}
	//remove heading "IAM token:  Bearer "
	token := string(dat)[19:]
	//fmt.Printf("\ntoken = %s", token)
	return token
}
