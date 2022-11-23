package pkg

import (
	"context"
	"fmt"
	ipamv1alpha1 "github.com/onmetal/ipam/api/v1alpha1"
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
	if err := ipamv1alpha1.AddToScheme(scheme.Scheme); err != nil {
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

func (k K8sClient) getInventoryUUIDByMac(mac, namespace string) (string, error) {

	mac = "machine.onmetal.de/mac-address-" + strings.ReplaceAll(mac, ":", "")
	var inventory inventoryv1alpha1.InventoryList
	err := k.Client.List(context.Background(), &inventory, client.InNamespace(namespace), client.MatchingLabels{mac: ""})
	if err != nil {
		err = errors.Wrapf(err, "Failed to list inventories in namespace %s", namespace)
		return "", err
	}

	var uuid string
	if len(inventory.Items) == 0 {
		return "", nil
	} else if len(inventory.Items) > 1 {
		return "", errors.New(fmt.Sprintf("Multiple inventories found for mac %s", mac))
	} else if len(inventory.Items) == 1 {
		uuid = inventory.Items[0].Spec.System.ID
	}

	log.Printf("Found Inventory UUID %s for MAC: %s", uuid, mac)
	return uuid, nil
}

func (k K8sClient) getIPByMac(mac, namespace string) (string, error) {
	var ips ipamv1alpha1.IPList
	err := k.Client.List(context.Background(),
		&ips,
		client.InNamespace(namespace),
		client.MatchingLabels{"mac": strings.ReplaceAll(mac, ":", "")})
	if err != nil {
		err = errors.Wrapf(err, "Failed to list IPAM IPs in namespace %s", namespace)
		return "", err
	}

	var ip string
	if len(ips.Items) == 0 {
		return "", errors.New(fmt.Sprintf("Mac %s is unknown", mac))
	} else if len(ips.Items) > 1 {
		return "", errors.New(fmt.Sprintf("Mac %s has more then one IPs", mac))
	} else if len(ips.Items) == 1 {
		ip = ips.Items[0].Spec.IP.String()
	}

	log.Printf("Found IPAM IP %s for MAC: %s", ip, mac)
	return ip, nil
}
