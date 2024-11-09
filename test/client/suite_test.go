/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gcustom"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	operatorclientset "github.com/sap/component-operator/pkg/client/clientset/versioned"
	operatorinformers "github.com/sap/component-operator/pkg/client/informers/externalversions"
	operatorv1alpha1informers "github.com/sap/component-operator/pkg/client/informers/externalversions/core.cs.sap.com/v1alpha1"
	operatorv1alpha1listers "github.com/sap/component-operator/pkg/client/listers/core.cs.sap.com/v1alpha1"
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

var clientset operatorclientset.Interface
var componentInformer operatorv1alpha1informers.ComponentInformer
var componentLister operatorv1alpha1listers.ComponentLister

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

	By("populating scheme")
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(apiregistrationv1.AddToScheme(scheme))
	operator.InitScheme(scheme)

	By("initializing client")
	cli, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	By("initializing generated client")
	clientset, err = operatorclientset.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	informerFactory := operatorinformers.NewSharedInformerFactory(clientset, 10*time.Minute)
	componentInformer = informerFactory.Core().V1alpha1().Components()

	componentLister = componentInformer.Lister()
	informerFactory.Start(ctx.Done())
	_ctx, _cancel := context.WithTimeout(ctx, 10*time.Second)
	defer _cancel()
	for _, ok := range informerFactory.WaitForCacheSync(_ctx.Done()) {
		Expect(ok).To(BeTrue())
	}
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

var _ = Describe("Test generated client package", func() {
	var namespace string
	var component1 *operatorv1alpha1.Component
	var component2 *operatorv1alpha1.Component
	var component3 *operatorv1alpha1.Component

	BeforeEach(func() {
		namespace = createNamespace()
		component1 = createComponent(namespace)
		component2 = createComponent(namespace)
		component3 = createComponent(namespace)
	})

	Describe("Test clientset", func() {
		It("should perform direct read", func() {
			component, err := clientset.CoreV1alpha1().Components(namespace).Get(ctx, component1.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			Expect(component).To(KubeEqual(component1))
		})

		It("should perform direct list", func() {
			componentList, err := clientset.CoreV1alpha1().Components(namespace).List(ctx, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())

			components := make([]*operatorv1alpha1.Component, len(componentList.Items))
			for i := 0; i < len(componentList.Items); i++ {
				components[i] = &componentList.Items[i]
			}
			Expect(components).To(ConsistOf(KubeEqual(component1), KubeEqual(component2), KubeEqual(component3)))
		})
	})

	Describe("Test lister", func() {
		It("should perform cached read", func() {
			Eventually(func(g Gomega) {
				component, err := componentLister.Components(namespace).Get(component1.Name)
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(component).To(KubeEqual(component1))
			}, "5s", "100ms").Should(Succeed())
		})

		It("should perform cached list", func() {
			Eventually(func(g Gomega) {
				components, err := componentLister.Components(namespace).List(labels.Everything())
				g.Expect(err).NotTo(HaveOccurred())

				g.Expect(components).To(ConsistOf(KubeEqual(component1), KubeEqual(component2), KubeEqual(component3)))
			}, "5s", "100ms").Should(Succeed())
		})
	})

	Describe("Test informer with event handler", func() {
		It("should receive events", func() {
			var components []*operatorv1alpha1.Component

			handler := cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj any) {
					component := obj.(*operatorv1alpha1.Component)
					if component.Namespace == namespace {
						components = append(components, component)
					}
				},
			}
			handlerRegistration, err := componentInformer.Informer().AddEventHandler(handler)
			defer componentInformer.Informer().RemoveEventHandler(handlerRegistration)
			Expect(err).NotTo(HaveOccurred())
			Eventually(handlerRegistration.HasSynced, "5s", "100ms").Should(BeTrue())

			Expect(components).To(ConsistOf(KubeEqual(component1), KubeEqual(component2), KubeEqual(component3)))

			component := createComponent(namespace)
			Eventually(func() *operatorv1alpha1.Component { return components[len(components)-1] }, "5s", "100ms").Should(KubeEqual(component))
		})
	})

})

// create namespace with random name
func createNamespace() string {
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "ns-"}}
	err := cli.Create(ctx, namespace)
	Expect(err).NotTo(HaveOccurred())
	return namespace.Name
}

// create minimalistic component with random name
func createComponent(namespace string) *operatorv1alpha1.Component {
	component := &operatorv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    namespace,
			GenerateName: "test-",
		},
	}
	err := cli.Create(ctx, component)
	Expect(err).NotTo(HaveOccurred())
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

// custom matcher to compare two Kubernetes objects, ignoring typemeta (unless unstructured, where typemeta must be there)
func KubeEqual(expected runtime.Object) OmegaMatcher {
	return gcustom.MakeMatcher(func(actual runtime.Object) (bool, error) {
		_, actualUnstructured := actual.(*unstructured.Unstructured)
		_, expectedUnstructured := expected.(*unstructured.Unstructured)
		if !actualUnstructured && !expectedUnstructured {
			actual = actual.DeepCopyObject()
			expected = expected.DeepCopyObject()
			actual.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{})
			expected.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{})
		}
		return reflect.DeepEqual(actual, expected), nil
	})
}
