/*
Copyright 2019 The Skaffold Authors

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

package diagnose

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSizeOfDockerContext(t *testing.T) {
	tests := []struct {
		description string
		dockerfile  string
		files       map[string]string
		expected    int64
		shouldErr   bool
	}{
		{
			description: "test size",
			dockerfile:  "Dockerfile",
			expected:    2048,
		},
		{
			description: "test size for a image with file",
			dockerfile:  "Dockerfile",
			files:       map[string]string{"foo": "foo"},
			expected:    3072,
		},
		{
			description: "dockerfile not found",
			dockerfile:  "Dockerfile.notfound",
			shouldErr:   true,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			tmpDir := t.NewTempDir().
				Write("Dockerfile", "FROM scratch").
				WriteFiles(test.files)

			dummyArtifact := &latest.Artifact{
				Workspace: tmpDir.Root(),
				ArtifactType: latest.ArtifactType{
					DockerArtifact: &latest.DockerArtifact{
						DockerfilePath: test.dockerfile,
					},
				},
			}

			actual, err := sizeOfDockerContext(dummyArtifact)

			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, actual)
		})
	}
}

func TestCheckArtifacts(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		tmpDir := t.NewTempDir().Write("Dockerfile", "FROM busybox")

		runCtx := &runcontext.RunContext{
			Cfg: latest.Pipeline{
				Build: latest.BuildConfig{
					Artifacts: []*latest.Artifact{{
						Workspace: tmpDir.Root(),
						ArtifactType: latest.ArtifactType{
							DockerArtifact: &latest.DockerArtifact{
								DockerfilePath: "Dockerfile",
							},
						},
					}},
				},
			},
		}
		err := CheckArtifacts(context.Background(), runCtx, ioutil.Discard)

		t.CheckNoError(err)
	})
}
