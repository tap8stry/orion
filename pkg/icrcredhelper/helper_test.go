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

import "testing"

// TestIsACRHelper tests whether URLs are detected as being ACR/MCR registry
// hosts, to determine whether the cred helper should fail fast.
func TestIsICRHelper(t *testing.T) {
	for _, c := range []struct {
		url  string
		want bool
	}{
		{"icr.io", true},
		{"au.icr.io", true},
		{"br.icr.io", true},
		{"ca.icr.io", true},
		{"de.icr.io", true},
		{"jp.icr.io", true},
		{"jp2.icr.io", true},
		{"uk.icr.io", true},
		{"us.icr.io", true},
		{"dd.icr.io", false},
		{"127.0.0.1:12345", false},
		{"localhost:12345", false},
		{"notaurl-)(*$@)(*@)(*", false},
	} {
		t.Run(c.url, func(t *testing.T) {
			got := IsICRRegistry(c.url)
			if got != c.want {
				t.Fatalf("got %t, want %t", got, c.want)
			}
		})
	}
}
