// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package client

import (
	"errors"
	"flag"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
	extconfig "github.com/steadybit/extension-auto-registration-kubernetes/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	informersCorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	listerCorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Client struct {
	pod struct {
		lister   listerCorev1.PodLister
		informer informersCorev1.PodIndexInformer
	}
	service struct {
		lister   listerCorev1.ServiceLister
		informer informersCorev1.ServiceIndexInformer
	}
}

func PrepareClient(stopCh <-chan struct{}) *Client {
	clientset := createClientset()
	result := checkPermissions(clientset)
	if result.HasErrors() {
		log.Fatal().Msg("Required permissions are missing. Exit now.")
	}

	return CreateClient(clientset, stopCh)
}

func createClientset() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err == nil {
		log.Info().Msgf("Extension is running inside a cluster, config found")
	} else if errors.Is(err, rest.ErrNotInCluster) {
		log.Info().Msgf("Extension is not running inside a cluster, try local .kube config")
		var kubeconfig *string
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		flag.Parse()
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}

	if err != nil {
		log.Fatal().Err(err).Msgf("Could not find kubernetes config")
	}

	config.UserAgent = "steadybit-extension-auto-registration-kubernetes"
	config.Timeout = time.Second * 10
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not create kubernetes client")
	}

	info, err := clientset.ServerVersion()
	if err != nil {
		log.Fatal().Err(err).Msgf("Could not fetch server version.")
	}

	log.Info().Msgf("Cluster connected! Kubernetes Server Version %+v", info)

	return clientset
}

// CreateClient is visible for testing
func CreateClient(clientset kubernetes.Interface, stopCh <-chan struct{}) *Client {
	client := &Client{}

	var factory informers.SharedInformerFactory
	if extconfig.Config.NamespaceFilter != "" {
		factory = informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithNamespace(extconfig.Config.NamespaceFilter))
	} else {
		factory = informers.NewSharedInformerFactory(clientset, 0)
	}

	var informerSyncList []cache.InformerSynced

	pods := factory.Core().V1().Pods()
	client.pod.informer = pods.TypedInformer()
	client.pod.lister = pods.Lister()
	informerSyncList = append(informerSyncList, client.pod.informer.HasSynced)
	if err := client.pod.informer.SetTransform(transformPod); err != nil {
		log.Fatal().Err(err).Msg("Failed to add pod transformer")
	}

	services := factory.Core().V1().Services()
	client.service.informer = services.TypedInformer()
	client.service.lister = services.Lister()
	informerSyncList = append(informerSyncList, client.service.informer.HasSynced)
	if err := client.service.informer.SetTransform(transformService); err != nil {
		log.Fatal().Err(err).Msg("Failed to add service transformer")
	}

	defer runtime.HandleCrash()
	go factory.Start(stopCh)

	log.Info().Msgf("Start Kubernetes cache sync.")
	if !cache.WaitForCacheSync(stopCh, informerSyncList...) {
		log.Fatal().Msg("Timed out waiting for caches to sync")
	}
	log.Info().Msgf("Kubernetes caches synced.")

	return client
}

func (c *Client) WatchPods(onAdd func(pod *corev1.Pod), onUpdate func(oldPod, newPod *corev1.Pod), onDelete func(pod informersCorev1.DeletedPod)) {
	if _, err := c.pod.informer.AddTypedEventHandler(informersCorev1.PodHandlerFuncs{
		AddFunc: func(pod *corev1.Pod) {
			log.Trace().Str("pod", pod.Name).Str("namespace", pod.Namespace).Msg("k8s pod added")
			onAdd(pod)
		},
		UpdateFunc: func(oldPod, newPod *corev1.Pod) {
			log.Trace().Str("pod", newPod.Name).Str("namespace", newPod.Namespace).Msg("k8s pod updated")
			onUpdate(oldPod, newPod)
		},
		DeleteFunc: func(pod informersCorev1.DeletedPod) {
			log.Trace().Str("pod", pod.GetName()).Str("namespace", pod.GetNamespace()).Msg("k8s pod deleted")
			onDelete(pod)
		},
	}); err != nil {
		log.Fatal().Msg("failed to add pod event handler")
	}
}

func (c *Client) WatchServices(onAdd func(service *corev1.Service), onUpdate func(oldService, newService *corev1.Service), onDelete func(service informersCorev1.DeletedService)) {
	if _, err := c.service.informer.AddTypedEventHandler(informersCorev1.ServiceHandlerFuncs{
		AddFunc: func(service *corev1.Service) {
			log.Trace().Str("service", service.Name).Str("namespace", service.Namespace).Msg("k8s service added")
			onAdd(service)
		},
		UpdateFunc: func(oldService, newService *corev1.Service) {
			log.Trace().Str("service", newService.Name).Str("namespace", newService.Namespace).Msg("k8s service updated")
			onUpdate(oldService, newService)
		},
		DeleteFunc: func(service informersCorev1.DeletedService) {
			log.Trace().Str("service", service.GetName()).Str("namespace", service.GetNamespace()).Msg("k8s service deleted")
			onDelete(service)
		},
	}); err != nil {
		log.Fatal().Msg("failed to add service event handler")
	}
}

func (c *Client) ServicesByPod(pod *corev1.Pod) []*corev1.Service {
	services, err := c.service.lister.Services(pod.Namespace).List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msg("Error while fetching services")
		return []*corev1.Service{}
	}

	podLabels := labels.Set(pod.Labels)
	var result []*corev1.Service

	for _, service := range services {
		if len(service.Spec.Selector) == 0 {
			continue
		}

		selector := labels.SelectorFromSet(service.Spec.Selector)
		if selector.Matches(podLabels) {
			result = append(result, service)
		}
	}
	return result
}

func (c *Client) PodsByService(service *corev1.Service) []*corev1.Pod {
	pods, err := c.pod.lister.Pods(service.Namespace).List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msg("Error while fetching pods")
		return []*corev1.Pod{}
	}

	if len(service.Spec.Selector) == 0 {
		return []*corev1.Pod{}
	}

	selector := labels.SelectorFromSet(service.Spec.Selector)
	var result []*corev1.Pod

	for _, pod := range pods {
		if selector.Matches(labels.Set(pod.Labels)) {
			result = append(result, pod)
		}
	}
	return result
}

func (c *Client) IsPodRunningAndReady(pod *corev1.Pod) bool {
	if pod == nil {
		return false
	}

	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
