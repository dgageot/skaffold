/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tag

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestEnvTemplateTagger_GenerateFullyQualifiedImageName(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		imageName string
		env       []string
		expected  string
	}{
		{
			name:      "empty env with digest (deprecated)",
			template:  "{{.IMAGE_NAME}}:{{.DIGEST}}",
			imageName: "foo",
			expected:  "foo",
		},
		{
			name:      "env",
			template:  "{{.FOO}}-{{.BAZ}}:latest",
			env:       []string{"FOO=BAR", "BAZ=BAT"},
			imageName: "foo",
			expected:  "BAR-BAT:latest",
		},
		{
			name:      "opts precedence",
			template:  "{{.IMAGE_NAME}}-{{.FROM_ENV}}:latest",
			env:       []string{"FROM_ENV=FOO", "IMAGE_NAME=BAT"},
			imageName: "image_name",
			expected:  "image_name-FOO:latest",
		},
		{
			name:      "digest algo hex (deprecated)",
			template:  "{{.IMAGE_NAME}}:{{.DIGEST_ALGO}}-{{.DIGEST_HEX}}",
			imageName: "foo",
			expected:  "foo",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			util.OSEnviron = func() []string {
				return test.env
			}

			c, err := NewEnvTemplateTagger(test.template)
			testutil.CheckError(t, false, err)

			got, err := c.GenerateFullyQualifiedImageName("", test.imageName)

			testutil.CheckErrorAndDeepEqual(t, false, err, test.expected, got)
		})
	}
}

func TestNewEnvTemplateTagger(t *testing.T) {
	tests := []struct {
		name      string
		template  string
		shouldErr bool
	}{
		{
			name:     "valid template",
			template: "{{.FOO}}",
		},
		{
			name:      "invalid template",
			template:  "{{.FOO",
			shouldErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEnvTemplateTagger(tt.template)
			testutil.CheckError(t, tt.shouldErr, err)
		})
	}
}
