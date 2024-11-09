/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	kyaml "sigs.k8s.io/yaml"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	"github.com/sap/component-operator/pkg/operator"
)

func TestOperator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator")
}

var testEnv *envtest.Environment
var cfg *rest.Config
var cli client.Client
var ctx context.Context
var cancel context.CancelFunc
var threads sync.WaitGroup
var tmpdir string
var repositoryUrl string
var repository *Repository

var _ = BeforeSuite(func() {
	var err error

	By("initializing")
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())
	tmpdir, err = os.MkdirTemp("", "")
	Expect(err).NotTo(HaveOccurred())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{"../../crds"},
	}
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = clientcmd.WriteToFile(*kubeConfigFromRestConfig(cfg), fmt.Sprintf("%s/kubeconfig", tmpdir))
	Expect(err).NotTo(HaveOccurred())
	fmt.Printf("A temporary kubeconfig for the envtest environment can be found here: %s/kubeconfig\n", tmpdir)

	By("create operator")
	operator := operator.NewWithOptions(operator.Options{EnableFlux: ref(false)})

	By("populating scheme")
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(apiregistrationv1.AddToScheme(scheme))
	operator.InitScheme(scheme)

	By("initializing client")
	cli, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	By("creating manager")
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: append(operator.GetUncacheableTypes(), &apiextensionsv1.CustomResourceDefinition{}, &apiregistrationv1.APIService{}),
			},
		},
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		HealthProbeBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	err = operator.Setup(mgr)
	Expect(err).NotTo(HaveOccurred())

	By("starting http repository")
	repositoryPort, err := getFreePort("127.0.0.1")
	Expect(err).NotTo(HaveOccurred())
	repositoryAddress := fmt.Sprintf("127.0.0.1:%d", repositoryPort)
	repositoryUrl = fmt.Sprintf("http://%s", repositoryAddress)
	repository = newRepository(scheme)
	threads.Add(1)
	go func() {
		defer threads.Done()
		defer GinkgoRecover()

		server := http.Server{Addr: repositoryAddress, Handler: repository}
		go func() {
			<-ctx.Done()
			server.Close()
		}()
		err = server.ListenAndServe()
		if err == http.ErrServerClosed {
			err = nil
		}
		Expect(err).NotTo(HaveOccurred())
	}()

	By("starting manager")
	threads.Add(1)
	go func() {
		defer threads.Done()
		defer GinkgoRecover()
		err := mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	threads.Wait()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	err = os.RemoveAll(tmpdir)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Deploy components", func() {
	var namespace string

	BeforeEach(func() {
		namespace = createNamespace()
	})

	It("should reconcile given component from http repository", func() {
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
		repository.Upload(namespace, cm)
		createComponentAndWait(namespace, 30*time.Second)
		cm.Namespace = namespace
		assertObject(cm)
	})
})

// create namespace with random name
func createNamespace() string {
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "ns-"}}
	err := cli.Create(ctx, namespace)
	Expect(err).NotTo(HaveOccurred())
	return namespace.Name
}

// assert that object exists
func assertObject(object client.Object) {
	err := cli.Get(ctx, apitypes.NamespacedName{Namespace: object.GetNamespace(), Name: object.GetName()}, object)
	Expect(err).NotTo(HaveOccurred())
}

// create minimalistic component with random name
func createComponent(namespace string) *operatorv1alpha1.Component {
	component := &operatorv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: "test-",
		},
		Spec: operatorv1alpha1.ComponentSpec{
			SourceRef: operatorv1alpha1.SourceReference{
				HttpRepository: &operatorv1alpha1.HttpRepository{
					Url: repositoryUrl + "/" + namespace,
				},
			},
		},
	}
	err := cli.Create(ctx, component)
	Expect(err).NotTo(HaveOccurred())
	return component
}

// create component and wait for it to get ready
func createComponentAndWait(namespace string, timeout time.Duration) *operatorv1alpha1.Component {
	component := createComponent(namespace)
	Eventually(func() error {
		if err := cli.Get(ctx, apitypes.NamespacedName{Namespace: component.Namespace, Name: component.Name}, component); err != nil {
			return err
		}
		if !component.IsReady() {
			return fmt.Errorf("not ready - retrying")
		}
		return nil
	}, timeout.String(), "1s").Should(Succeed())
	return component
}

// convert rest.Config into kubeconfig
func kubeConfigFromRestConfig(restConfig *rest.Config) *clientcmdapi.Config {
	apiConfig := clientcmdapi.NewConfig()

	apiConfig.Clusters["envtest"] = clientcmdapi.NewCluster()
	cluster := apiConfig.Clusters["envtest"]
	cluster.Server = restConfig.Host
	cluster.CertificateAuthorityData = restConfig.CAData

	apiConfig.AuthInfos["envtest"] = clientcmdapi.NewAuthInfo()
	authInfo := apiConfig.AuthInfos["envtest"]
	authInfo.ClientKeyData = restConfig.KeyData
	authInfo.ClientCertificateData = restConfig.CertData

	apiConfig.Contexts["envtest"] = clientcmdapi.NewContext()
	context := apiConfig.Contexts["envtest"]
	context.Cluster = "envtest"
	context.AuthInfo = "envtest"

	apiConfig.CurrentContext = "envtest"

	return apiConfig
}

// get free port
func getFreePort(address string) (uint16, error) {
	for i := 0; i < 10; i++ {
		port := 0
		addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(address, strconv.Itoa(port)))
		if err != nil {
			return 0, err
		}
		listener, err := net.ListenTCP("tcp", addr)
		listener.Close()
		if err != nil {
			continue
		}
		port = listener.Addr().(*net.TCPAddr).Port
		if port != 0 {
			// TODO: the following cast is potentially unsafe (however no port numbers outside the 0-65535 range should occur)
			return uint16(port), nil
		}
	}
	return 0, fmt.Errorf("unable to find free port")
}

// get ref to variable
func ref[T any](x T) *T {
	return &x
}

// http repository
type Repository struct {
	scheme *runtime.Scheme
	data   map[string][]byte
}

func newRepository(scheme *runtime.Scheme) *Repository {
	return &Repository{
		scheme: scheme,
		data:   make(map[string][]byte),
	}
}

func (r *Repository) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	data, ok := r.data[req.URL.Path]
	if !ok {
		http.Error(rw, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	digest := sha256.Sum256(data)
	rw.Header().Set("etag", hex.EncodeToString(digest[:]))
	rw.Write(data)
}

func (r *Repository) Upload(path string, resources ...runtime.Object) error {
	if path == "" || path[0] != '/' {
		path = "/" + path
	}
	rawBuffer := &bytes.Buffer{}
	for _, resource := range resources {
		gvk, err := apiutil.GVKForObject(resource, r.scheme)
		if err != nil {
			return err
		}
		resource = resource.DeepCopyObject()
		resource.GetObjectKind().SetGroupVersionKind(gvk)

		raw, err := kyaml.Marshal(resource)
		if err != nil {
			return err
		}
		rawBuffer.WriteString("---\n")
		rawBuffer.Write(raw)
	}
	zippedBuffer := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(zippedBuffer)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "resources.yaml",
		Mode: 0644,
		Size: int64(rawBuffer.Len()),
	}); err != nil {
		return err
	}
	if _, err := tarWriter.Write(rawBuffer.Bytes()); err != nil {
		return err
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}
	r.data[path] = zippedBuffer.Bytes()
	return nil
}
