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
		informer cache.SharedIndexInformer
	}
	service struct {
		lister   listerCorev1.ServiceLister
		informer cache.SharedIndexInformer
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
	client.pod.informer = pods.Informer()
	client.pod.lister = pods.Lister()
	informerSyncList = append(informerSyncList, client.pod.informer.HasSynced)
	if err := client.pod.informer.SetTransform(transformPod); err != nil {
		log.Fatal().Err(err).Msg("Failed to add pod transformer")
	}

	services := factory.Core().V1().Services()
	client.service.informer = services.Informer()
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

func (c *Client) WatchPods(add func(pod *corev1.Pod), update func(old *corev1.Pod, new *corev1.Pod), delete func(pod *corev1.Pod)) {
	if _, err := c.pod.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			add(pod)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod := oldObj.(*corev1.Pod)
			newPod := newObj.(*corev1.Pod)
			update(oldPod, newPod)
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			delete(pod)
		},
	}); err != nil {
		log.Fatal().Msg("failed to add pod event handler")
	}
}

func (c *Client) ServicesByPod(pod *corev1.Pod) []*corev1.Service {
	services, err := c.service.lister.Services(pod.Namespace).List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("Error while fetching services")
		return []*corev1.Service{}
	}
	var result []*corev1.Service
	for _, service := range services {
		match := service.Spec.Selector != nil
		for key, value := range service.Spec.Selector {
			if value != pod.ObjectMeta.Labels[key] {
				match = false
			}
		}
		if match {
			result = append(result, service)
		}
	}
	return result
}

func (c *Client) IsPodRunningAndReady(pod *corev1.Pod) bool {
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
