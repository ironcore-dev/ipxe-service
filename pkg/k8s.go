package pkg

import (
	"context"
	inventoryv1alpha1 "github.com/onmetal/metal-api/apis/inventory/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"strings"
)

type K8sClient struct {
	Client client.Client
}

func NewK8sClient() K8sClient {
	if err := inventoryv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal("Unable to add registered types inventory to client scheme: %s", err)
	}

	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		log.Fatal("Failed to create a client: ", err)
	}
	return K8sClient{
		Client: cl,
	}
}

func (k K8sClient) getSecret(name, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	err := k.Client.Get(context.Background(), client.ObjectKeyFromObject(secret), secret)
	if err != nil {
		log.Printf("Failed to get Secret %s in Namespace %s: %s", name, namespace, err)
		return nil, err
	}

	return secret, nil
}

func (k K8sClient) getConfigMag(name, namespace string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	err := k.Client.Get(context.Background(), client.ObjectKeyFromObject(configMap), configMap)
	if err != nil {
		log.Printf("Failed to get ConfigMap %s in Namespace %s: %s", name, namespace, err)
		return nil, err
	}

	return configMap, nil
}

func (k K8sClient) getIPsFromInventory(uuid, namespace string) ([]string, error) {

	inventory := &inventoryv1alpha1.Inventory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uuid,
			Namespace: namespace,
		},
	}
	err := k.Client.Get(context.Background(), client.ObjectKeyFromObject(inventory), inventory)
	if err != nil {
		err = errors.Wrapf(err, "Failed to get inventory %s in namespace %s", uuid, namespace)
		return []string{}, err
	}

	ips := []string{}
	for label, _ := range inventory.Labels {
		if strings.HasPrefix(label, "machine.onmetal.de/ip-address-") {
			ip := strings.ReplaceAll(label, "machine.onmetal.de/ip-address-", "")
			ip = strings.ReplaceAll(ip, "_", ":")
			ips = append(ips, ip)
		}
	}

	return ips, nil
}
