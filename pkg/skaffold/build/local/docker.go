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

package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/pkg/errors"
)

func (b *Builder) buildDocker(ctx context.Context, out io.Writer, workspace string, a *latest.DockerArtifact, fqn string) (string, error) {
	if b.cfg.UseDockerCLI || b.cfg.UseBuildkit {
		dockerfilePath, err := docker.NormalizeDockerfilePath(workspace, a.DockerfilePath)
		if err != nil {
			return "", errors.Wrap(err, "normalizing dockerfile path")
		}

		args := []string{"build", workspace, "--file", dockerfilePath, "-t", fqn}
		args = append(args, docker.GetBuildArgs(a)...)

		cmd := exec.CommandContext(ctx, "docker", args...)
		if b.cfg.UseBuildkit {
			cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
		}
		cmd.Stdout = out
		cmd.Stderr = out

		if err := util.RunCmd(cmd); err != nil {
			return "", errors.Wrap(err, "running build")
		}
	} else {
		if _, err := b.localDocker.Build(ctx, out, workspace, a, fqn); err != nil {
			return "", errors.Wrap(err, "running build")
		}
	}

	if b.pushImages {
		digest, err := b.localDocker.Push(ctx, out, fqn)
		if err != nil {
			return "", errors.Wrap(err, "pushing image")
		}

		return fmt.Sprintf("%s@%s", fqn, digest), nil
	}

	return fqn, nil
}
