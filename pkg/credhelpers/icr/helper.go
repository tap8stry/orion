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

package icr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
)

// ibmcloud cloud registries by region
var icrregions = [9]string{
	"icr.io",
	"au.icr.io",
	"br.icr.io",
	"ca.icr.io",
	"de.icr.io",
	"jp.icr.io",
	"jp2.icr.io",
	"uk.icr.io",
	"us.icr.io",
}

const (
	tokenUsername  = "iambearer"            //user must be "iambearer"
	ibmcloudConfig = ".bluemix/config.json" //where token is stored
	iamTokenEnvVar = "IBMCLOUD_IAM_TOEKN"   //environment variable for ibmcloud iam token
	Instruction    = "run following commands to get/set your ibmcloud iam token:\n\t1. ibmcloud login -a https://cloud.ibm.com --apikey <your api key> -r <region> \n\t2. ibmcloud iam oauth-tokens\n\t3. export IBMCLOUD_IAM_TOEKN=<token>\n "
)

type ICRCredHelper struct {
}

type IBMCloudConfig struct {
	IAMToken string `json:"IAMToken"`
}

func NewICRCredentialsHelper() credentials.Helper {
	return &ICRCredHelper{}
}

func (a ICRCredHelper) Add(_ *credentials.Credentials) error {
	return errors.New("list is not implemented")
}

func (a ICRCredHelper) Delete(_ string) error {
	return errors.New("list is not implemented")
}

func IsICRRegistry(input string) bool {
	serverURL, err := url.Parse("https://" + input)
	if err != nil {
		return false
	}
	return validateICR(serverURL.Hostname())
}

func (a ICRCredHelper) Get(serverURL string) (string, string, error) {
	if !IsICRRegistry(serverURL) {
		return "", "", errors.New("serverURL does not point to an IBMCloud Container Registry")
	}
	//try get token from env variable first. if not available try ibmcloud config.json
	token := getTokenFromEnv(iamTokenEnvVar)
	if len(token) == 0 {
		token = getTokenFromConfig()
		if len(token) == 0 {
			return "", "", errors.New("no token found in env variables or ibmcloud config file")
		}
	}
	return tokenUsername, token, nil
}

func (a ICRCredHelper) List() (map[string]string, error) {
	return nil, errors.New("list is not implemented")
}

func validateICR(target string) bool {
	for _, reg := range icrregions {
		if strings.EqualFold(target, reg) {
			return true
		}
	}
	return false
}

func getTokenFromEnv(envVar string) string {
	var token string
	token = os.Getenv(envVar)
	return token
}

func getTokenFromConfig() string {
	//get iam bearer token from ibmcloud config.json
	var thisuser *user.User
	var file *os.File
	var dat []byte
	var icConfig IBMCloudConfig
	var err error
	if thisuser, err = user.Current(); err != nil {
		fmt.Printf("\nerror getting home directory: %s", err.Error())
		return ""
	}
	fp := filepath.Join(thisuser.HomeDir, ibmcloudConfig)
	if file, err = os.Open(fp); err != nil {
		fmt.Printf("\nerror opening ibmcloud config file %q: %s", err.Error(), fp)
		return ""
	}
	if dat, err = ioutil.ReadAll(file); err != nil {
		fmt.Printf("\nerror reading ibmcloud config file %q: %s", err.Error(), fp)
		return ""
	}
	if err = json.Unmarshal(dat, &icConfig); err != nil {
		fmt.Printf("\nerror unmarshal ibmcloud config %q: %s", err.Error(), fp)
		return ""
	}
	if len(icConfig.IAMToken) < 7 { //empty token
		return ""
	}
	return icConfig.IAMToken[7:] //remove "Bearer "
}
