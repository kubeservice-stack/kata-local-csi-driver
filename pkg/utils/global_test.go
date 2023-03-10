/*
Copyright 2023 The Kubernetes Authors.

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

package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalENV(t *testing.T) {
	assert := assert.New(t)
	os.Setenv("KUBELET_ROOT_DIR", "/tmp1")
	os.Setenv("TLS_MOUNT_DIR", "/tmp2")

	assert.Equal(KubeletRootDir, "/var/lib/kubelet")
	assert.Equal(MountPathWithTLS, "/tls")
}
