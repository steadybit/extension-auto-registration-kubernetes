package autoregistration

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-auto-registration-kubernetes/client"
	"github.com/steadybit/extension-auto-registration-kubernetes/config"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type AutoRegistration struct {
	httpClient                          *resty.Client
	k8sClient                           *client.Client
	discoveredExtensions                *sync.Map
	isDirty                             atomic.Bool
	agentRegistrationInterval           time.Duration
	agentRegistrationIntervalAfterError time.Duration
	matchLabels                         config.Labels
	matchLabelsExclude                  config.Labels
}

func UpdateAgentExtensions(httpClient *resty.Client, k8sClient *client.Client) {
	registrator := AutoRegistration{
		httpClient:                          httpClient,
		k8sClient:                           k8sClient,
		discoveredExtensions:                &sync.Map{},
		agentRegistrationInterval:           config.Config.AgentRegistrationInterval,
		agentRegistrationIntervalAfterError: config.Config.AgentRegistrationIntervalAfterError,
		matchLabels:                         config.Config.MatchLabels,
		matchLabelsExclude:                  config.Config.MatchLabelsExclude,
		isDirty:                             atomic.Bool{},
	}

	registrator.syncRegistrations()
	k8sClient.WatchPods(registrator.processAddedPod, registrator.processUpdatedPod, registrator.processDeletedPod)
}

func (r *AutoRegistration) processAddedPod(pod *corev1.Pod) {
	log.Trace().Str("pod", pod.Name).Str("namespace", pod.Namespace).Msg("k8s pod added")
	if r.k8sClient.IsPodRunningAndReady(pod) {
		extensions := r.toExtensionConfigs(pod)
		if len(extensions) > 0 {
			r.discoveredExtensions.Store(r.key(pod), extensions)
			log.Debug().Str("pod", pod.Name).Str("namespace", pod.Namespace).Int("count", len(extensions)).Msg("Adding extension registration.")
			r.isDirty.Store(true)
		}
	}
}

func (r *AutoRegistration) processUpdatedPod(_ *corev1.Pod, new *corev1.Pod) {
	log.Trace().Str("pod", new.Name).Str("namespace", new.Namespace).Msg("k8s pod updated")
	if r.k8sClient.IsPodRunningAndReady(new) {
		extensions := r.toExtensionConfigs(new)
		if len(extensions) > 0 {
			r.discoveredExtensions.Store(r.key(new), extensions)
			log.Debug().Str("pod", new.Name).Str("namespace", new.Namespace).Int("count", len(extensions)).Msg("Adding extension registration.")
			r.isDirty.Store(true)
		}
	} else {
		value, loaded := r.discoveredExtensions.LoadAndDelete(r.key(new))
		if loaded {
			v := value.([]extensionConfigAO)
			log.Debug().Str("pod", new.Name).Str("namespace", new.Namespace).Int("count", len(v)).Msg("Remove extension registration.")
			r.isDirty.Store(true)
		}
	}
}
func (r *AutoRegistration) processDeletedPod(pod *corev1.Pod) {
	log.Trace().Str("pod", pod.Name).Str("namespace", pod.Namespace).Msg("k8s pod deleted")
	value, loaded := r.discoveredExtensions.LoadAndDelete(r.key(pod))
	if loaded {
		v := value.([]extensionConfigAO)
		log.Debug().Str("pod", pod.Name).Str("namespace", pod.Namespace).Int("count", len(v)).Msg("Remove extension registration.")
		r.isDirty.Store(true)
	}
}

func workloadMatchesSelector(podLabel map[string]string, matchLabel []config.Label) bool {
	for _, label := range matchLabel {
		if value, exists := podLabel[label.Key]; !exists || value != label.Value {
			return false
		}
	}
	return true
}

func (r *AutoRegistration) toExtensionConfigs(pod *corev1.Pod) []extensionConfigAO {
	result := make([]extensionConfigAO, 0)

	if len(r.matchLabels) != 0 && !workloadMatchesSelector(pod.Labels, r.matchLabels) {
		log.Trace().Str("pod", pod.Name).Str("namespace", pod.Namespace).Msg("Exclude candidate because it does not match matchLabels.")
		return result
	}
	if len(r.matchLabelsExclude) != 0 && workloadMatchesSelector(pod.Labels, r.matchLabelsExclude) {
		log.Trace().Str("pod", pod.Name).Str("namespace", pod.Namespace).Msg("Exclude candidate because it matches matchLabelsExclude.")
		return result
	}

	podAnnotations := r.getExtensionAnnotations(pod.Annotations)
	if len(podAnnotations) > 0 {
		podIP := pod.Status.PodIP
		if podIP == "" {
			log.Warn().Str("pod", pod.Name).Str("namespace", pod.Namespace).Msg("Pod has extension annotations but no IP. Ignoring.")
			return result
		}
		for _, annotation := range podAnnotations {
			url := fmt.Sprintf("%s://%s", annotation.Protocol, podIP)
			if annotation.Port > 0 {
				url += ":" + strconv.Itoa(annotation.Port)
			}
			url += annotation.Path
			result = append(result, extensionConfigAO{
				Url:             url,
				RestrictedPorts: r.getAdditionalPortsOfPod(pod),
				RestrictedIps:   []string{podIP},
			})
		}
	} else {
		for _, service := range r.k8sClient.ServicesByPod(pod) {
			log.Debug().Str("pod", pod.Name).Str("namespace", pod.Namespace).Str("service", service.Name).Msg("Found service for pod.")
			serviceAnnotations := r.getExtensionAnnotations(service.Annotations)
			if len(serviceAnnotations) > 0 {
				restrictedPorts := make(map[int]string)
				restrictedIps := make([]string, 0)
				for _, s := range service.Spec.Ports {
					restrictedPorts[int(s.Port)] = "ServicePort"
				}
				mergeMaps(restrictedPorts, r.getAdditionalPortsOfPod(pod))
				for _, ingress := range service.Status.LoadBalancer.Ingress {
					if ingress.IP != "" {
						restrictedIps = append(restrictedIps, ingress.IP)
					}
				}
				if pod.Status.PodIP != "" {
					restrictedIps = append(restrictedIps, pod.Status.PodIP)
				}
				for _, annotation := range serviceAnnotations {
					url := fmt.Sprintf("%s://%s.%s.svc.cluster.local", annotation.Protocol, service.Name, service.Namespace)
					if annotation.Port > 0 {
						url += ":" + strconv.Itoa(annotation.Port)
					}
					url = url + annotation.Path
					result = append(result, extensionConfigAO{
						Url:             url,
						RestrictedIps:   restrictedIps,
						RestrictedPorts: restrictedPorts,
					})
				}
			}
		}
	}
	return result
}

func (r *AutoRegistration) getExtensionAnnotations(annotations map[string]string) []ExtensionAnnotation {
	if annotations == nil {
		return []ExtensionAnnotation{}
	}
	keys := []string{"steadybit.com/extension-auto-registration", "steadybit.com/extension-auto-discovery"}
	for _, key := range keys {
		if val, ok := annotations[key]; ok {
			return r.parseAnnotationJSON(val)
		}
	}
	return []ExtensionAnnotation{}
}

func (r *AutoRegistration) parseAnnotationJSON(value string) []ExtensionAnnotation {
	var extAnnotations ExtensionAnnotations
	err := json.Unmarshal([]byte(value), &extAnnotations)
	if err != nil {
		log.Err(err).Str("value", value).Msg("Failed to parse extension annotation. Ignoring.")
		return []ExtensionAnnotation{}
	}
	return extAnnotations.Extensions
}

func (r *AutoRegistration) key(pod *corev1.Pod) string {
	return pod.Namespace + "/" + pod.Name
}

func (r *AutoRegistration) getAdditionalPortsOfPod(pod *corev1.Pod) map[int]string {
	if len(pod.Spec.Containers) == 0 {
		return map[int]string{}
	}

	additionalPorts := make(map[int]string)
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			additionalPorts[int(port.ContainerPort)] = "ContainerPort"
		}
		if container.LivenessProbe != nil && container.LivenessProbe.HTTPGet != nil {
			additionalPorts[int(container.LivenessProbe.HTTPGet.Port.IntVal)] = "LivenessProbe"
		}
		if container.ReadinessProbe != nil && container.ReadinessProbe.HTTPGet != nil {
			additionalPorts[int(container.ReadinessProbe.HTTPGet.Port.IntVal)] = "ReadinessProbe"
		}
	}

	if !r.containsValue(additionalPorts, "ReadinessProbe") && !r.containsValue(additionalPorts, "LivenessProbe") {
		additionalPorts[8081] = "Defaulted HealthPort"
	}
	return additionalPorts
}

func (r *AutoRegistration) containsValue(m map[int]string, value string) bool {
	for _, v := range m {
		if v == value {
			return true
		}
	}
	return false
}

func (r *AutoRegistration) syncRegistrations() {
	if !r.isDirty.Load() {
		time.AfterFunc(r.agentRegistrationInterval, r.syncRegistrations)
		log.Trace().Msgf("No changes detected, waiting for %s before next check.", r.agentRegistrationInterval)
		return
	}

	var errGet, errRemove, errAdd error
	currentRegistrations, errGet := getCurrentRegistrations(r.httpClient)
	if errGet == nil {
		discoveredExtensions := make([]extensionConfigAO, 0)
		r.discoveredExtensions.Range(func(key, value any) bool {
			v := value.([]extensionConfigAO)
			discoveredExtensions = append(discoveredExtensions, v...)
			return true
		})
		r.isDirty.Store(false)
		errRemove = removeMissingRegistrations(r.httpClient, currentRegistrations, discoveredExtensions)
		errAdd = addNewRegistrations(r.httpClient, currentRegistrations, discoveredExtensions)
	}

	if (errGet != nil) || (errRemove != nil) || (errAdd != nil) {
		r.isDirty.Store(true)
		log.Info().Msgf("Retry in %s", r.agentRegistrationIntervalAfterError)
		time.AfterFunc(r.agentRegistrationIntervalAfterError, r.syncRegistrations)
	} else {
		log.Debug().Msg("Registrations synced successfully.")
		time.AfterFunc(r.agentRegistrationInterval, r.syncRegistrations)
	}
}

func mergeMaps(dest, src map[int]string) {
	for key, value := range src {
		dest[key] = value
	}
}
