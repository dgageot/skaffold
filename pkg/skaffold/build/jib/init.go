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

package jib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// For testing
var (
	ValidateJibConfigFunc = ValidateJibConfig
)

// Jib holds information about a Jib project
type Jib struct {
	BuilderName string `json:"-"`
	Image       string `json:"image,omitempty"`
	FilePath    string `json:"path,omitempty"`
	Project     string `json:"project,omitempty"`
}

// Name returns the name of the builder
func (j Jib) Name() string {
	return j.BuilderName
}

// Describe returns the initBuilder's string representation, used when prompting the user to choose a builder.
func (j Jib) Describe() string {
	if j.Project != "" {
		return fmt.Sprintf("%s (%s, %s)", j.BuilderName, j.Project, j.FilePath)
	}
	return fmt.Sprintf("%s (%s)", j.BuilderName, j.FilePath)
}

// CreateArtifact creates an Artifact to be included in the generated Build Config
func (j Jib) UpdateArtifact(a *latest.Artifact) {
	a.ArtifactType = latest.ArtifactType{
		JibArtifact: &latest.JibArtifact{
			Project: j.Project,
		},
	}

	workspace := filepath.Dir(j.FilePath)
	if workspace != "." {
		a.Workspace = workspace
	}
}

// ConfiguredImage returns the target image configured by the builder, or empty string if no image is configured
func (j Jib) ConfiguredImage() string {
	return j.Image
}

// Path returns the path to the build definition
func (j Jib) Path() string {
	return j.FilePath
}

// BuilderConfig contains information about inferred Jib configurations
type jibJSON struct {
	Image   string `json:"image"`
	Project string `json:"project"`
}

// ValidateJibConfig checks if a file is a valid Jib configuration. Returns the list of Config objects corresponding to each Jib project built by the file, or nil if Jib is not configured.
func ValidateJibConfig(path string) []Jib {
	// Determine whether maven or gradle
	var builderType PluginType
	var executable, wrapper, taskName, searchString string
	switch {
	case strings.HasSuffix(path, "pom.xml"):
		builderType = JibMaven
		executable = "mvn"
		wrapper = "mvnw"
		searchString = "<artifactId>jib-maven-plugin</artifactId>"
		taskName = "jib:_skaffold-init"
	case strings.HasSuffix(path, "build.gradle"), strings.HasSuffix(path, "build.gradle.kts"):
		builderType = JibGradle
		executable = "gradle"
		wrapper = "gradlew"
		searchString = "com.google.cloud.tools.jib"
		taskName = "_jibSkaffoldInit"
	default:
		return nil
	}

	// Search for indication of Jib in build file before proceeding
	if content, err := ioutil.ReadFile(path); err != nil || !strings.Contains(string(content), searchString) {
		return nil
	}

	// Run Jib's skaffold init task/goal to check if Jib is configured
	if wrapperExecutable, err := util.AbsFile(filepath.Dir(path), wrapper); err == nil {
		executable = wrapperExecutable
	}
	cmd := exec.Command(executable, taskName, "-q")
	cmd.Dir = filepath.Dir(path)
	stdout, err := util.RunCmdOut(cmd)
	if err != nil {
		return nil
	}

	// Parse Jib output. Multiple JSON strings may be printed for multi-project/multi-module setups.
	matches := regexp.MustCompile(`BEGIN JIB JSON\r?\n({.*})`).FindAllSubmatch(stdout, -1)
	if len(matches) == 0 {
		return nil
	}

	results := make([]Jib, len(matches))
	for i, match := range matches {
		// Escape windows path separators
		line := bytes.Replace(match[1], []byte(`\`), []byte(`\\`), -1)
		parsedJSON := jibJSON{}
		if err := json.Unmarshal(line, &parsedJSON); err != nil {
			logrus.Warnf("failed to parse jib json: %s", err.Error())
			return nil
		}

		results[i] = Jib{BuilderName: PluginName(builderType), Image: parsedJSON.Image, FilePath: path, Project: parsedJSON.Project}
	}
	return results
}
