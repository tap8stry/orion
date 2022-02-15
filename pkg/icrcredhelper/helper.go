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
	"errors"
	"net/url"
	"os"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/tap8stry/orion/pkg/common"
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
	tokenUsername = "iambearer" //user must be "iambearer"
)

type ICRCredHelper struct {
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
		return "", "", errors.New("serverURL does not refer to IBMCloud Container Registry")
	}

	//get iam bearer token
	apikey := os.Getenv(common.APIKeyEnv)
	token := GetToken(apikey)
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
