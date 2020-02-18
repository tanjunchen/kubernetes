/*
Copyright 2015 The Kubernetes Authors.

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

package e2e

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	clientset "k8s.io/client-go/kubernetes"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	commonutils "k8s.io/kubernetes/test/e2e/common"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/auth"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"k8s.io/kubernetes/test/e2e/framework/testfiles"

	"github.com/onsi/ginkgo"
)

const (
	serverStartTimeout = framework.PodStartTimeout + 3*time.Minute
)

/**
E2E 测试可以添加标签，如下所示
无	测试可以快速（5m以内）完成，支持并行测试，具有一致性
[Slow]	运行时间超过5分钟
[Serial]	不支持和其它测试并行执行
[Disruptive]	可能影响（例如重启组件、Taint节点）不是该测试自己创建的工作负载。任何 Disruptive 测试自动是Serial的
[Flaky]	标记测试中的问题难以短期修复。这种测试默认情况下不会运行，除非使用 focus/skip 参数
[Feature:.+]	如果一个测试运行/处理非核心功能，因此需要排除出标准测试套件，使用此标签。
[LinuxOnly]	需要使用 Linux 特有的特性
 */

/**
此外 任何测试都必须归属于某个 SIG 并具有对应的 [sig-<name>]标签。
每个 e2e 的子包都在 framework.go 中 SIGDescribe 函数, 来添加此标签。
测试可以具有多个标签，使用空格分隔即可。
*/

// 声明一个 ginkgo.Describe 块，自动添加 [k8s.io] 标签
var _ = framework.KubeDescribe("[Feature:Example]", func() {
	//   创建一个新的 Framework 对象，自动提供：
	//   BeforeEach：创建 K8S 客户端、创建命名空间、启动资源用量收集器、指标收集器
	//   AfterEach：调用 cleanupHandle、删除命名空间
	f := framework.NewDefaultFramework("examples")

	var c clientset.Interface
	var ns string
	// 自己可以添加额外的 Setup/Teardown 块
	ginkgo.BeforeEach(func() {
		// 获取客户端、使用的命名空间
		c = f.ClientSet
		ns = f.Namespace.Name

		// this test wants powerful permissions.  Since the namespace names are unique, we can leave this
		// lying around so we don't have to race any caches
		// 在命名空间级别绑定 RBAC 权限，给 default 服务账号授权
		err := auth.BindClusterRoleInNamespace(c.RbacV1(), "edit", f.Namespace.Name,
			rbacv1.Subject{Kind: rbacv1.ServiceAccountKind, Namespace: f.Namespace.Name, Name: "default"})
		framework.ExpectNoError(err)

		// 等待操作完成
		err = auth.WaitForAuthorizationUpdate(c.AuthorizationV1(),
			serviceaccount.MakeUsername(f.Namespace.Name, "default"),
			f.Namespace.Name, "create", schema.GroupResource{Resource: "pods"}, true)
		framework.ExpectNoError(err)
	})

	// 嵌套的Describe
	framework.KubeDescribe("Liveness", func() {
		// 第一个 Spec：健康测试检查失败的 Pod 能否自动重启
		ginkgo.It("liveness pods should be automatically restarted", func() {
			test := "test/fixtures/doc-yaml/user-guide/liveness"

			// 读取文件，Go Template形式，并解析为YAML资源清单
			execYaml := readFile(test, "exec-liveness.yaml.in")
			httpYaml := readFile(test, "http-liveness.yaml.in")
			nsFlag := fmt.Sprintf("--namespace=%v", ns)

			// 调用 kubectl 来创建资源
			framework.RunKubectlOrDieInput(execYaml, "create", "-f", "-", nsFlag)
			framework.RunKubectlOrDieInput(httpYaml, "create", "-f", "-", nsFlag)

			// Since both containers start rapidly, we can easily run this test in parallel.
			// 并行测试
			var wg sync.WaitGroup
			passed := true
			// 此函数检查发生了重启
			checkRestart := func(podName string, timeout time.Duration) {
				// 等待 Pod 就绪
				err := e2epod.WaitForPodNameRunningInNamespace(c, podName, ns)
				framework.ExpectNoError(err)
				// 轮询直到重启次数大于 0
				for t := time.Now(); time.Since(t) < timeout; time.Sleep(framework.Poll) {
					pod, err := c.CoreV1().Pods(ns).Get(podName, metav1.GetOptions{})
					framework.ExpectNoError(err, fmt.Sprintf("getting pod %s", podName))
					stat := podutil.GetExistingContainerStatus(pod.Status.ContainerStatuses, podName)
					framework.Logf("Pod: %s, restart count:%d", stat.Name, stat.RestartCount)
					if stat.RestartCount > 0 {
						framework.Logf("Saw %v restart, succeeded...", podName)
						wg.Done()
						return
					}
				}
				framework.Logf("Failed waiting for %v restart! ", podName)
				passed = false
				wg.Done()
			}
			// By 用于添加一段文档说明
			ginkgo.By("Check restarts")

			// Start the "actual test", and wait for both pods to complete.
			// If 2 fail: Something is broken with the test (or maybe even with liveness).
			// If 1 fails: Its probably just an error in the examples/ files themselves.
			// 检查两个 Pod
			wg.Add(2)
			for _, c := range []string{"liveness-http", "liveness-exec"} {
				go checkRestart(c, 2*time.Minute)
			}
			wg.Wait()
			if !passed {
				framework.Failf("At least one liveness example failed.  See the logs above.")
			}
		})
	})

	// 第二个 Spec，测试 Pod 能读取 Secret
	framework.KubeDescribe("Secret", func() {
		ginkgo.It("should create a pod that reads a secret", func() {
			test := "test/fixtures/doc-yaml/user-guide/secrets"
			secretYaml := readFile(test, "secret.yaml")
			podYaml := readFile(test, "secret-pod.yaml.in")

			nsFlag := fmt.Sprintf("--namespace=%v", ns)
			podName := "secret-test-pod"

			ginkgo.By("creating secret and pod")
			// 创建一个 Secret，以及会读取此 Secret 并打印的 Pod
			framework.RunKubectlOrDieInput(secretYaml, "create", "-f", "-", nsFlag)
			framework.RunKubectlOrDieInput(podYaml, "create", "-f", "-", nsFlag)
			// 等待 Pod 退出
			err := e2epod.WaitForPodNoLongerRunningInNamespace(c, podName, ns)
			framework.ExpectNoError(err)

			ginkgo.By("checking if secret was read correctly")
			// 检查 Pod 日志
			_, err = framework.LookForStringInLog(ns, "secret-test-pod", "test-container", "value-1", serverStartTimeout)
			framework.ExpectNoError(err)
		})
	})

	// 第三个 Spec, 测试 Downward API
	framework.KubeDescribe("Downward API", func() {
		ginkgo.It("should create a pod that prints his name and namespace", func() {
			test := "test/fixtures/doc-yaml/user-guide/downward-api"
			podYaml := readFile(test, "dapi-pod.yaml.in")
			nsFlag := fmt.Sprintf("--namespace=%v", ns)
			podName := "dapi-test-pod"

			ginkgo.By("creating the pod")
			framework.RunKubectlOrDieInput(podYaml, "create", "-f", "-", nsFlag)
			err := e2epod.WaitForPodNoLongerRunningInNamespace(c, podName, ns)
			framework.ExpectNoError(err)

			ginkgo.By("checking if name and namespace were passed correctly")
			_, err = framework.LookForStringInLog(ns, podName, "test-container", fmt.Sprintf("MY_POD_NAMESPACE=%v", ns), serverStartTimeout)
			framework.ExpectNoError(err)
			_, err = framework.LookForStringInLog(ns, podName, "test-container", fmt.Sprintf("MY_POD_NAME=%v", podName), serverStartTimeout)
			framework.ExpectNoError(err)
		})
	})
})

func readFile(test, file string) string {
	from := filepath.Join(test, file)
	return commonutils.SubstituteImageName(string(testfiles.ReadOrDie(from)))
}
