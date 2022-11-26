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
	"net"
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

func (k K8sClient) getMacFromIP(clientIP, namespace string) (string, error) {
	if getIPVersion(clientIP) == "ipv6" {
		ip := net.ParseIP(clientIP)
		clientIP = getLongIPv6(ip)
	}

	var ips ipamv1alpha1.IPList
	err := k.Client.List(context.Background(),
		&ips,
		client.InNamespace(namespace),
		client.MatchingLabels{"ip": strings.ReplaceAll(clientIP, ":", "_")})
	if err != nil {
		err = errors.Wrapf(err, "Failed to list IPAM IPs in namespace %s", namespace)
		return "", err
	}

	var mac string
	if len(ips.Items) == 0 {
		return "", errors.New(fmt.Sprintf("IP %s is unknown", clientIP))
	} else if len(ips.Items) > 1 {
		return "", errors.New(fmt.Sprintf("More than one IP %s found", clientIP))
	} else if len(ips.Items) == 1 {
		macLabel, exists := ips.Items[0].Labels["mac"]
		if !exists {
			return "", errors.New(fmt.Sprintf("No Mac was found for IP %s", clientIP))
		}
		mac = macLabel
	}

	log.Printf("Mac %s for IPAM IP %s found", mac, clientIP)
	return mac, nil
}

func (k K8sClient) getInventory(uuid, namespace string) (*inventoryv1alpha1.Inventory, error) {

	inventory := &inventoryv1alpha1.Inventory{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uuid,
			Namespace: namespace,
		},
	}
	err := k.Client.Get(context.Background(), client.ObjectKeyFromObject(inventory), inventory)
	if err != nil {
		err = errors.Wrapf(err, "Failed to get inventory in namespace %s", namespace)
		return nil, err
	}

	log.Printf("Found Inventory for UUID %s", uuid)
	return inventory, nil
}
