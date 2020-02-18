/*
Copyright 2017 The Kubernetes Authors.

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

package node

import "k8s.io/kubernetes/test/e2e/framework"

/**
此外 任何测试都必须归属于某个 SIG 并具有对应的 [sig-<name>]标签。
每个 e2e 的子包都在 framework.go 中 SIGDescribe 函数, 来添加此标签。
测试可以具有多个标签，使用空格分隔即可。
*/

// SIGDescribe annotates the test with the SIG label.
func SIGDescribe(text string, body func()) bool {
	return framework.KubeDescribe("[sig-node] "+text, body)
}
