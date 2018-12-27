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

package kaniko

import (
	"context"
	"fmt"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/kaniko/sources"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Build builds a list of artifacts with Kaniko.
func (b *Builder) Build(ctx context.Context, out io.Writer, tagger tag.Tagger, artifacts []*latest.Artifact) ([]build.Artifact, error) {
	teardown, err := b.setupSecret(out)
	if err != nil {
		return nil, errors.Wrap(err, "setting up secret")
	}
	defer teardown()

	return build.InParallel(ctx, out, tagger, artifacts, b.buildArtifactWithKaniko)
}

func (b *Builder) buildArtifactWithKaniko(ctx context.Context, out io.Writer, artifact *latest.Artifact, fqn string) (string, error) {
	s := sources.Retrieve(b.KanikoBuild)
	context, err := s.Setup(ctx, out, artifact, fqn)
	if err != nil {
		return "", errors.Wrap(err, "setting up build context")
	}
	defer s.Cleanup(ctx)

	client, err := kubernetes.GetClientset()
	if err != nil {
		return "", errors.Wrap(err, "")
	}

	args := []string{
		fmt.Sprintf("--dockerfile=%s", artifact.DockerArtifact.DockerfilePath),
		fmt.Sprintf("--context=%s", context),
		fmt.Sprintf("--destination=%s", fqn),
		fmt.Sprintf("-v=%s", logLevel().String())}
	args = append(args, b.AdditionalFlags...)
	args = append(args, docker.GetBuildArgs(artifact.DockerArtifact)...)

	if b.Cache != nil {
		args = append(args, "--cache=true")
		if b.Cache.Repo != "" {
			args = append(args, fmt.Sprintf("--cache-repo=%s", b.Cache.Repo))
		}
	}

	pods := client.CoreV1().Pods(b.Namespace)
	p, err := pods.Create(s.Pod(args))
	if err != nil {
		return "", errors.Wrap(err, "creating kaniko pod")
	}
	defer func() {
		if err := pods.Delete(p.Name, &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		}); err != nil {
			logrus.Fatalf("deleting pod: %s", err)
		}
	}()

	if err := s.ModifyPod(ctx, p); err != nil {
		return "", errors.Wrap(err, "modifying kaniko pod")
	}

	waitForLogs := streamLogs(out, p.Name, pods)

	if err := kubernetes.WaitForPodComplete(ctx, pods, p.Name, b.timeout); err != nil {
		return "", errors.Wrap(err, "waiting for pod to complete")
	}

	waitForLogs()

	return docker.FullRemoteReference(fqn)
}
