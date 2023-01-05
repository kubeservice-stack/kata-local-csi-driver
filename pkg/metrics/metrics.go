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

package metrics

import (
	"io"
	"strings"
	"sync"
	"time"

	"github.com/kubeservice-stack/kata-local-csi-driver/pkg/utils"

	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/uber-go/tally"
	promreporter "github.com/uber-go/tally/prometheus"
)

var onceEnable sync.Once

// metrics 发布器
type TallyScope struct {
	Reporter promreporter.Reporter
	Scope    tally.Scope
	Closer   io.Closer
}

// TODO 默认metrics Scope
var DefaultTallyScope *TallyScope

// 创建
func NewTallyScope(prefix string, tags map[string]string, interval time.Duration) *TallyScope {
	onceEnable.Do(func() {
		DefaultRegistry().Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		DefaultRegistry().Register(collectors.NewGoCollector())
	})

	r := promreporter.NewReporter(promreporter.Options{})
	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		Prefix:         prefix,
		Tags:           tags,
		CachedReporter: r,
		Separator:      promreporter.DefaultSeparator,
	}, interval*time.Second) // interval * time.Second

	return &TallyScope{Scope: scope, Closer: closer, Reporter: r}
}

// 进程失败销毁
func (t *TallyScope) Destroy() error {
	return t.Closer.Close()
}

func init() {
	r := strings.NewReplacer(".", "_", "-", "_")
	DefaultTallyScope = NewTallyScope(r.Replace(utils.ProvisionerName), map[string]string{}, 5) // 默认填充 defaultTallyScope
}
