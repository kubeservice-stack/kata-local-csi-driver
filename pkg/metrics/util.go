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
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally"
)

var DefaultTallyBuckets = tally.ValueBuckets{.01, .05, .1, .2, .5, .8, .9, 1, 5, 10, 15, 30, 60, 90, 120}

var (
	ErrMetricsInitRegistryError = errors.New("metrics: can not init metrics registry")
	ErrMetricsDoNotExist        = errors.New("metrics: metrics do not exists")
	ErrMetricsIsDuplicated      = errors.New("metrics: metric is duplicated")
)

// 默认Registerer
var metricsRegisterer = prometheus.DefaultRegisterer

// 默认prometheus.DefaultRegisterer, 没有特殊需要，不用使用NewRegisterer
func DefaultRegistry() prometheus.Registerer {
	return metricsRegisterer
}
