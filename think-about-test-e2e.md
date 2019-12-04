Hello,oomichi:

After the [#79686]https://github.com/kubernetes/kubernetes/issues/79686 is basically completed,I think there are some gomega.* references that can be encapsulated and integrated.Just like `gomega.Expect(foo).To(gomega.Equal(bar))`.

I found that some wrapper function in test/e2e/framework,such as:

    ```
    func ExpectEqual(actual interface{}, extra interface{}, explain ...interface{})
    func ExpectNotEqual(actual interface{}, extra interface{}, explain ...interface{})
    func ExpectError(err error, explain ...interface{})
    func ExpectNoError(err error, explain ...interface{})
    func Failf(format string, args ...interface{})
    func log(level string, format string, args ...interface{})
    func Fail(msg string, callerSkip ...int) 
    other functions
    ```
Or we can continue to encapsulate some simple gomega.* in test/e2e/ folder. We can replace this code with framework.WrapperFunction(function, paramaters...) which is more readable and easy to be understood what the code does.
The following is my rough survey calling a function in `gomega.*` form
    ```
    form     the number of summary
    gomega.  599
    gomega.BeNumerically 44 
    gomega.HaveKey 23
    gomega.BeEmpty() 31
    gomega.Eventually(func(),...).Should(...)  140
    gomega.Consistently(func(),...).Should(...)  14
    gomega.Succeed() 32
    gomega.BeNil()  165
    other format  ...

    For Example:
    gomega.Consistently(func() (int, error) {
        logs, err := kubelogs.GetAllLogs(logPath)
        if err != nil {
            return 0, err
        }
        return len(logs), nil
    }, rotationConsistentlyTimeout, rotationPollInterval).Should(gomega.BeNumerically("<=", testContainerLogMaxFiles),
    "should never exceed max file limit")

    gomega.Consistently(func() error {
        err := checkMirrorPodRunning(f.ClientSet, mirrorPodName, ns)
        if err == nil {
            framework.Logf("mirror pod %q is running", mirrorPodName)
        } else {
            framework.Logf(err.Error())
        }
        return err
    }, time.Minute*8, time.Second*4).ShouldNot(gomega.HaveOccurred())

    gomega.Eventually(func() error {
        return validateOOMScoreAdjSetting(pid, -999)
    }, 5*time.Minute, 30*time.Second).Should(gomega.BeNil())

    gomega.Eventually(func() error {
    ngPids, err = getPidsForProcess("nginx", "")
        if err != nil {
            return fmt.Errorf("failed to get list of nginx process pids: %v", err)
        }
        for _, pid := range ngPids {
            if err := validateOOMScoreAdjSetting(pid, -998); err != nil {
                return err
            }
        }
        return nil
    }, 2*time.Minute, time.Second*4).Should(gomega.BeNil())
    ```

Do you have similar ideas? Or what is your suggestion? How should we encapsulate it?
my thought:
    ```
    // ExpectHaveKey succeeds if actual is a map with the passed in key.
    func ExpectHaveKey(actual interface{}, key interface{}) {
    gomega.Expect(actual).To(gomega.HaveKey(key))
    }

    // ExpectNotEmpty succeeds if actual is empty.	Actual must be of type string, array, map, chan, or slice.
    func ExpectNotEmpty(actual interface{}, explain ...interface{}) {
    gomega.Expect(actual).NotTo(gomega.BeEmpty(), explain...)
    }

    // ExpectEmpty succeeds if actual is empty.	Actual must be of type string, array, map, chan, or slice.
    func ExpectEmpty(actual interface{}, explain ...interface{}) {
    gomega.Expect(actual).To(gomega.BeEmpty(), explain...)
    }

    // BeNumerically performs numerical assertions in a type-agnostic way.
    func BeNumerically(actual interface{}, comparator string, compareTo ...interface{}) {
    gomega.Expect(actual).To(gomega.BeNumerically(comparator, compareTo))
    }

    // ExpectToSucceed passes if actual is a nil error
    func ExpectToSucceed(actual interface{}) {
    gomega.Expect(actual).To(gomega.Succeed())
    }

    // ExpectSholdSucceed passes if actual is a nil error
    func ExpectSholdSucceed(actual interface{}) {
    gomega.Expect(actual).Should(gomega.Succeed())
    }
    ```

Maybe the above wrapper function can be replaced with a function that has been encapsulated, and there is still a lot of work to be done, after all, the [gomega](http://onsi.github.io/gomega/#benumericallycomparator-string-compareto-interface) and [ginkgo](http://onsi.github.io/ginkgo/#documenting-complex-its-by) test frameworks involve too many functions and parameters.

If you have better ideas or opinions, please tell me, thank you!

                            tanjunchen(githubId)