package e2e_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/test/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsclient "k8s.io/client-go/kubernetes/typed/apps/v1"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	interval           = time.Second * 2
	timeout            = time.Second * 60
	imageName          = "docker.io/nfvpe/sriov-device-plugin"
	dsName             = "kube-sriov-device-plugin"
	serviceAccountName = "sriov-device-plugin"
	defaultImageTag    = "v3.3"
)

var (
	master         *string
	kubeConfigPath *string
	testNs         *string
	testCmName     *string
	testNodeName   *string
	cs             *ClientSet
	ac             *appsclient.AppsV1Client

	pfNameForTest            *string
	numOfPfNetdev            *int
	numOfPfVfio              *int
	numOfVfNetdev            *int
	numOfVfVfio              *int
	numOfVfioVfForSelectedPf *int

	imageTag string
)

type ClientSet struct {
	coreclient.CoreV1Interface
}

func init() {
	if home := homedir.HomeDir(); home != "" {
		kubeConfigPath = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "path to your kubeconfig file")
	} else {
		kubeConfigPath = flag.String("kubeconfig", "", "require absolute path to your kubeconfig file")
	}
	master = flag.String("master", "", "Address of Kubernetes API server")
	testNs = flag.String("testnamespace", "kube-system", "namespace for testing")
	testCmName = flag.String("testcmname", "sriovdp-config", "name for test ConfigMap")
	testNodeName = flag.String("testnodename", "kind-control-plane", "name for test node")

	pfNameForTest = flag.String("pfnamefortest", "ens785f2", "Name of PF to use with pfDevice selector")
	numOfPfNetdev = flag.Int("numofpfnetdev", 2, "Number of PFs with kernel driver (e.g. i40e)")
	numOfPfVfio = flag.Int("numofpfvfio", 1, "Number of PFs with vfio-pci driver")
	numOfVfNetdev = flag.Int("numofvfnetdev", 4, "Number of VFs with kernel driver (e.g. iavf)")
	numOfVfVfio = flag.Int("numofvfvfio", 2, "Number of VFs with vfio-pci driver")
	numOfVfioVfForSelectedPf = flag.Int("numofvfiovfforselectedpf", 2,
							   "Number of VFs with vfio-pci driver enabled on slected PF")
}

func TestSriovTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SR-IOV DP E2E suite")
}

var _ = BeforeSuite(func(done Done) {
	cfg, err := clientcmd.BuildConfigFromFlags(*master, *kubeConfigPath)
	Expect(err).Should(BeNil())

	cs = &ClientSet{}
	cs.CoreV1Interface = coreclient.NewForConfigOrDie(cfg)

	ac = &appsclient.AppsV1Client{}
	ac = appsclient.NewForConfigOrDie(cfg)

	imageTag = os.Getenv("GITHUB_SHA")

	if imageTag == "" {
		imageTag = defaultImageTag
	}

	_ = util.DeleteServiceAccount(cs, serviceAccountName, *testNs)
	_ = util.DeleteDpDaemonset(ac, dsName, *testNs)

	err = util.CreateServiceAccount(cs, serviceAccountName, *testNs)
	Expect(err).To(BeNil())
	err = util.CreateDpDaemonset(ac, dsName, *testNs, imageName, imageTag)
	Expect(err).To(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	err := util.DeleteServiceAccount(cs, serviceAccountName, *testNs)
	Expect(err).To(BeNil())
	err = util.DeleteDpDaemonset(ac, dsName, *testNs)
	Expect(err).To(BeNil())
})
