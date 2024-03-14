// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package pkg

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	ipamv1alpha1 "github.com/ironcore-dev/ipam/api/ipam/v1alpha1"
	inventoryv1alpha4 "github.com/ironcore-dev/metal/apis/metal/v1alpha4"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg     *rest.Config
	testEnv *envtest.Environment
	ipxe    IPXE
	ctx     context.Context

	uuid               = "f2175eb4-e203-11ec-b5d5-3a68dd76b473"
	badUUID            = "b7960880-e203-11ec-b278-3a68dd76b3ef"
	emptyInventoryUUID = "94925a7e-d7e8-11ec-9bb5-3a68dd71f463"
	validIP1           = "fd00:0da8:fff6:3302::b:1"
	validIP2           = "fd00:0da8:fff6:3302::b:2"
	badIP              = "fd00:0da8:fff6:3302::f:1"
	namespace          = "metal-api-system"
)

func TestIPXEService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "iPXE Service Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	By("bootstrapping test environment")

	scheme := runtime.NewScheme()
	ctx = context.Background()

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}

	inventoryv1alpha4.SchemeBuilder.Register(func(scheme *runtime.Scheme) error {
		scheme.AddKnownTypes(inventoryv1alpha4.SchemeGroupVersion,
			&inventoryv1alpha4.Inventory{},
			&inventoryv1alpha4.InventoryList{},
		)
		return nil
	})

	ipamv1alpha1.SchemeBuilder.Register(
		&ipamv1alpha1.Network{},
		&ipamv1alpha1.NetworkList{},
		&ipamv1alpha1.Subnet{},
		&ipamv1alpha1.SubnetList{},
		&ipamv1alpha1.IP{},
		&ipamv1alpha1.IPList{},
	)

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	Expect(inventoryv1alpha4.AddToScheme(scheme)).NotTo(HaveOccurred())
	Expect(ipamv1alpha1.AddToScheme(scheme)).NotTo(HaveOccurred())
	Expect(corev1.AddToScheme(scheme)).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme
	k8sClient := NewK8sClient(cfg, client.Options{Scheme: scheme})
	Expect(k8sClient).ToNot(BeNil())

	conf := GetConf("../config/samples/config.yaml")
	ipxe = IPXE{
		Config:    conf,
		K8sClient: k8sClient,
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err = ipxe.K8sClient.Client.Create(ctx, ns)
	Expect(err).NotTo(HaveOccurred(), "failed to create metal-api-system Namespace")

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// SetupTestData will set up a testing environment.
// This includes:
// * creating metal-api-system Namespace
// * creating IPAM Network, Subnet and IP
// * creating Inventory, ConfigMap and Secret for f2175eb4-e203-11ec-b5d5-3a68dd76b473
// Call this function at the start of each of your tests.
func SetupTestData(ctx context.Context) {
	networkYaml, err := os.ReadFile("../config/samples/ipam/network.yaml")
	Expect(err).NotTo(HaveOccurred())
	network := &ipamv1alpha1.Network{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(networkYaml), 100).Decode(network)
	Expect(err).NotTo(HaveOccurred())

	subnetYaml, err := os.ReadFile("../config/samples/ipam/subnet.yaml")
	Expect(err).NotTo(HaveOccurred())
	subnet := &ipamv1alpha1.Subnet{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(subnetYaml), 100).Decode(subnet)
	Expect(err).NotTo(HaveOccurred())

	ip1Yaml, err := os.ReadFile("../config/samples/ipam/ip1.yaml")
	Expect(err).NotTo(HaveOccurred())
	ip1 := &ipamv1alpha1.IP{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(ip1Yaml), 100).Decode(ip1)
	Expect(err).NotTo(HaveOccurred())

	ip2Yaml, err := os.ReadFile("../config/samples/ipam/ip2.yaml")
	Expect(err).NotTo(HaveOccurred())
	ip2 := &ipamv1alpha1.IP{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(ip2Yaml), 100).Decode(ip2)
	Expect(err).NotTo(HaveOccurred())

	inventoryYaml, err := os.ReadFile("../config/samples/inventory/f2175eb4-e203-11ec-b5d5-3a68dd76b473.yaml")
	Expect(err).NotTo(HaveOccurred())
	inventory := &inventoryv1alpha4.Inventory{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(inventoryYaml), 100).Decode(inventory)
	Expect(err).NotTo(HaveOccurred())

	emptyInventory := &inventoryv1alpha4.Inventory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      emptyInventoryUUID,
			Namespace: namespace,
		},
		Spec: inventoryv1alpha4.InventorySpec{
			Host: &inventoryv1alpha4.HostSpec{
				Name: "",
			},
		},
	}

	configMapContent, err := os.ReadFile("../config/samples/configmap/ipxe-f2175eb4-e203-11ec-b5d5-3a68dd76b473")
	Expect(err).NotTo(HaveOccurred())
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ipxe-f2175eb4-e203-11ec-b5d5-3a68dd76b473",
			Namespace: namespace,
		},
		Data: map[string]string{
			"boot": string(configMapContent),
		},
	}

	secretContent, err := os.ReadFile("../config/samples/secret/ipxe-f2175eb4-e203-11ec-b5d5-3a68dd76b473")
	Expect(err).NotTo(HaveOccurred())
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ipxe-f2175eb4-e203-11ec-b5d5-3a68dd76b473",
			Namespace: namespace,
		},
		StringData: map[string]string{
			"ignition-default": string(secretContent),
		},
	}

	kubeconfigSecretYaml, err := os.ReadFile("../config/samples/secret/kubeconfig-inventory-94925a7e-d7e8-11ec-9bb5-3a68dd71f463.yaml")
	Expect(err).NotTo(HaveOccurred())
	kubeconfigSecret := &corev1.Secret{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(kubeconfigSecretYaml), 100).Decode(kubeconfigSecret)
	Expect(err).NotTo(HaveOccurred())

	BeforeEach(func() {
		err = ipxe.K8sClient.Client.Create(ctx, network.DeepCopy())
		Expect(err).NotTo(HaveOccurred(), "failed to create Network")
		err = ipxe.K8sClient.Client.Create(ctx, subnet.DeepCopy())
		Expect(err).NotTo(HaveOccurred(), "failed to create Subnet")
		err = ipxe.K8sClient.Client.Create(ctx, ip1.DeepCopy())
		Expect(err).NotTo(HaveOccurred(), "failed to create IP 1")
		err = ipxe.K8sClient.Client.Create(ctx, ip2.DeepCopy())
		Expect(err).NotTo(HaveOccurred(), "failed to create IP 2")
		err = ipxe.K8sClient.Client.Create(ctx, inventory.DeepCopy())
		Expect(err).NotTo(HaveOccurred(), "failed to create Inventory")
		err = ipxe.K8sClient.Client.Create(ctx, emptyInventory.DeepCopy())
		Expect(err).NotTo(HaveOccurred(), "failed to create empty Inventory")
		err = ipxe.K8sClient.Client.Create(ctx, configMap.DeepCopy())
		Expect(err).NotTo(HaveOccurred(), "failed to create ConfigMap")
		err = ipxe.K8sClient.Client.Create(ctx, secret.DeepCopy())
		Expect(err).NotTo(HaveOccurred(), "failed to create Secret")
		err = ipxe.K8sClient.Client.Create(ctx, kubeconfigSecret.DeepCopy())
		Expect(err).NotTo(HaveOccurred(), "failed to create kubeconfig Secret")
	})

	AfterEach(func() {
		err = ipxe.K8sClient.Client.Delete(ctx, network)
		Expect(err).NotTo(HaveOccurred(), "failed to delete Network")
		err = ipxe.K8sClient.Client.Delete(ctx, subnet)
		Expect(err).NotTo(HaveOccurred(), "failed to delete Subnet")
		err = ipxe.K8sClient.Client.Delete(ctx, ip1)
		Expect(err).NotTo(HaveOccurred(), "failed to delete IP 1")
		err = ipxe.K8sClient.Client.Delete(ctx, ip2)
		Expect(err).NotTo(HaveOccurred(), "failed to delete IP 2")
		err = ipxe.K8sClient.Client.Delete(ctx, inventory)
		Expect(err).NotTo(HaveOccurred(), "failed to delete Inventory")
		err = ipxe.K8sClient.Client.Delete(ctx, emptyInventory)
		Expect(err).NotTo(HaveOccurred(), "failed to delete empty Inventory")
		err = ipxe.K8sClient.Client.Delete(ctx, configMap)
		Expect(err).NotTo(HaveOccurred(), "failed to delete ConfigMap")
		err = ipxe.K8sClient.Client.Delete(ctx, secret)
		Expect(err).NotTo(HaveOccurred(), "failed to delete Secret")
		err = ipxe.K8sClient.Client.Delete(ctx, kubeconfigSecret)
		Expect(err).NotTo(HaveOccurred(), "failed to delete kubeconfig Secret")
	})
}
