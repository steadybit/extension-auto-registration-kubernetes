package client

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func transformPod(i interface{}) (interface{}, error) {
	//pod.Status.Phase
	//pod.Status.Conditions
	//pod.Status.PodIP
	//pod.ObjectMeta.Labels
	//pod.Extensions
	//pod.Name
	//pod.Namespace
	//pod.Spec.Containers
	if pod, ok := i.(*corev1.Pod); ok {
		pod.ObjectMeta = metav1.ObjectMeta{
			Name:        pod.Name,
			Namespace:   pod.Namespace,
			Labels:      pod.Labels,
			Annotations: pod.Annotations,
		}
		newPodSpec := corev1.PodSpec{
			Containers: make([]corev1.Container, 0, len(pod.Spec.Containers)),
		}
		for _, container := range pod.Spec.Containers {
			newPodSpec.Containers = append(newPodSpec.Containers, corev1.Container{
				Ports:          container.Ports,
				LivenessProbe:  container.LivenessProbe,
				ReadinessProbe: container.ReadinessProbe,
			})
		}
		pod.Spec = newPodSpec
		pod.Status = corev1.PodStatus{
			Phase:      pod.Status.Phase,
			Conditions: pod.Status.Conditions,
			PodIP:      pod.Status.PodIP,
		}
		return pod, nil
	}
	return i, nil
}

func transformService(i interface{}) (interface{}, error) {
	//service.Extensions
	//service.Name
	//service.Namespace
	//service.Spec.Selector
	//service.Spec.Ports
	//service.Status.LoadBalancer
	if s, ok := i.(*corev1.Service); ok {
		s.ObjectMeta = metav1.ObjectMeta{
			Name:        s.Name,
			Namespace:   s.Namespace,
			Annotations: s.Annotations,
		}
		s.Spec = corev1.ServiceSpec{
			Selector: s.Spec.Selector,
			Ports:    s.Spec.Ports,
		}
		s.Status = corev1.ServiceStatus{
			LoadBalancer: s.Status.LoadBalancer,
		}
		return s, nil
	}
	return i, nil
}
