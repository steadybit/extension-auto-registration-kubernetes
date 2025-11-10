package autoregistration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/steadybit/extension-auto-registration-kubernetes/client"
	"github.com/steadybit/extension-auto-registration-kubernetes/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestAutoRegistration_should_add_pods(t *testing.T) {
	discoveredExtensions := &sync.Map{}
	type args struct {
		pod                *corev1.Pod
		service            *corev1.Service
		matchLabels        config.Labels
		matchLabelsExclude config.Labels
	}
	tests := []struct {
		name                       string
		args                       args
		assertDiscoveredExtensions func(t *testing.T)
		assertAgentRegistrations   func(t *testing.T)
	}{
		{
			name: "should add daemonset pod without service",
			args: args{
				pod: getTestPod(nil),
			},
			assertDiscoveredExtensions: func(t *testing.T) {
				value, ok := discoveredExtensions.Load("default/test-pod")
				assert.True(t, ok, "Pod should be added to discoveredExtensions")
				extensions := value.([]extensionConfigAO)
				assert.Equal(t, 1, len(extensions), "There should be one extension registered")
				assert.Equal(t, "http://192.168.1.1:8080", extensions[0].Url, "Extension URL should match")
			},
			assertAgentRegistrations: func(t *testing.T) {
				assert.Len(t, AddedExtensions, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://192.168.1.1:8080\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\"},\"restrictedIps\":[\"192.168.1.1\"]}", AddedExtensions[0])
			},
		},
		{
			name: "should add daemonset pod without service via deprecated annotation",
			args: args{
				pod: getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{
						"steadybit.com/extension-auto-discovery": `{"extensions":[{"port":8080,"protocol":"http"}]}`,
					}
				}),
			},
			assertDiscoveredExtensions: func(t *testing.T) {
				value, ok := discoveredExtensions.Load("default/test-pod")
				assert.True(t, ok, "Pod should be added to discoveredExtensions")
				extensions := value.([]extensionConfigAO)
				assert.Equal(t, 1, len(extensions), "There should be one extension registered")
				assert.Equal(t, "http://192.168.1.1:8080", extensions[0].Url, "Extension URL should match")
			},
			assertAgentRegistrations: func(t *testing.T) {
				assert.Len(t, AddedExtensions, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://192.168.1.1:8080\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\"},\"restrictedIps\":[\"192.168.1.1\"]}", AddedExtensions[0])
			},
		},
		{
			name: "should add deployment pod with service",
			args: args{
				pod: getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}),
				service: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service",
						Namespace: "default",
						Annotations: map[string]string{
							"steadybit.com/extension-auto-registration": `{"extensions":[{"port":8085,"protocol":"http"}]}`,
						},
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{
							"app": "extension-xyz",
						},
						Ports: []corev1.ServicePort{
							{
								Port: 8085,
							},
						},
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{
									IP: "555.555.555.555",
								},
							},
						},
					},
				},
			},
			assertDiscoveredExtensions: func(t *testing.T) {
				value, ok := discoveredExtensions.Load("default/test-pod")
				assert.True(t, ok, "Pod should be added to discoveredExtensions")
				extensions := value.([]extensionConfigAO)
				assert.Equal(t, 1, len(extensions), "There should be one extension registered")
				assert.Equal(t, "http://test-service.default.svc.cluster.local:8085", extensions[0].Url, "Extension URL should match")
			},
			assertAgentRegistrations: func(t *testing.T) {
				assert.Len(t, AddedExtensions, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://test-service.default.svc.cluster.local:8085\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\",\"8085\":\"ServicePort\"},\"restrictedIps\":[\"555.555.555.555\",\"192.168.1.1\"]}", AddedExtensions[0])
			},
		},
		{
			name: "should ignore pod without annotations",
			args: args{
				pod: getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}),
			},
			assertDiscoveredExtensions: func(t *testing.T) {
				_, ok := discoveredExtensions.Load("default/test-pod")
				assert.False(t, ok, "Nothing should not be added to discoveredExtensions")
			},
			assertAgentRegistrations: func(t *testing.T) {
				assert.Empty(t, AddedExtensions, "Nothing should be registered")
			},
		},
		{
			name: "should ignore pod without ip",
			args: args{
				pod: getTestPod(func(p *corev1.Pod) {
					p.Status.PodIP = ""
				}),
			},
			assertDiscoveredExtensions: func(t *testing.T) {
				_, ok := discoveredExtensions.Load("default/test-pod")
				assert.False(t, ok, "Nothing should not be added to discoveredExtensions")
			},
			assertAgentRegistrations: func(t *testing.T) {
				assert.Empty(t, AddedExtensions, "Nothing should be registered")
			},
		},
		{
			name: "should ignore not running pod",
			args: args{
				pod: getTestPod(func(p *corev1.Pod) {
					p.Status.Phase = "Pending"
				}),
			},
			assertDiscoveredExtensions: func(t *testing.T) {
				_, ok := discoveredExtensions.Load("default/test-pod")
				assert.False(t, ok, "Nothing should not be added to discoveredExtensions")
			},
			assertAgentRegistrations: func(t *testing.T) {
				assert.Empty(t, AddedExtensions, "Nothing should be registered")
			},
		},
		{
			name: "should ignore not ready pod",
			args: args{
				pod: getTestPod(func(p *corev1.Pod) {
					p.Status.Conditions = []corev1.PodCondition{}
				}),
			},
			assertDiscoveredExtensions: func(t *testing.T) {
				_, ok := discoveredExtensions.Load("default/test-pod")
				assert.False(t, ok, "Nothing should not be added to discoveredExtensions")
			},
			assertAgentRegistrations: func(t *testing.T) {
				assert.Empty(t, AddedExtensions, "Nothing should be registered")
			},
		},
		{
			name: "should ignore pod not matching matchLabels",
			args: args{
				pod: getTestPod(nil),
				matchLabels: config.Labels{
					{
						Key:   "app",
						Value: "extension-abc",
					},
				},
			},
			assertDiscoveredExtensions: func(t *testing.T) {
				_, ok := discoveredExtensions.Load("default/test-pod")
				assert.False(t, ok, "Nothing should not be added to discoveredExtensions")
			},
			assertAgentRegistrations: func(t *testing.T) {
				assert.Empty(t, AddedExtensions, "Nothing should be registered")
			},
		},
		{
			name: "should ignore pod matching matchLabelsExclude",
			args: args{
				pod: getTestPod(nil),
				matchLabelsExclude: config.Labels{
					{
						Key:   "app",
						Value: "extension-xyz",
					},
				},
			},
			assertDiscoveredExtensions: func(t *testing.T) {
				_, ok := discoveredExtensions.Load("default/test-pod")
				assert.False(t, ok, "Nothing should not be added to discoveredExtensions")
			},
			assertAgentRegistrations: func(t *testing.T) {
				assert.Empty(t, AddedExtensions, "Nothing should be registered")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := createMockAgent()
			defer agent.Close()
			httpClient := resty.New()
			httpClient.BaseURL = agent.URL

			stopCh := make(chan struct{})
			defer close(stopCh)
			discoveredExtensions.Clear()
			k8sclient, k8stestclient := getTestClient(stopCh)
			if tt.args.service != nil {
				_, err := k8stestclient.CoreV1().Services(tt.args.service.Namespace).Create(context.Background(), tt.args.service, metav1.CreateOptions{})
				assert.NoError(t, err, "Service creation should succeed")
				time.Sleep(100 * time.Millisecond)
			}
			r := &AutoRegistration{
				httpClient:                httpClient,
				k8sClient:                 k8sclient,
				discoveredExtensions:      discoveredExtensions,
				agentRegistrationInterval: 1 * time.Second,
				matchLabels:               tt.args.matchLabels,
				matchLabelsExclude:        tt.args.matchLabelsExclude,
			}
			r.processAddedPod(tt.args.pod)
			r.syncRegistrations()
			assert.False(t, r.isDirty.Load(), "isDirty should be false after sync")
			tt.assertDiscoveredExtensions(t)
			MU.RLock()
			tt.assertAgentRegistrations(t)
			MU.RUnlock()
		})
	}
}

func getTestPod(modifier func(p *corev1.Pod)) *corev1.Pod {
	newPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "extension-xyz",
			},
			Annotations: map[string]string{
				"steadybit.com/extension-auto-registration": `{"extensions":[{"port":8080,"protocol":"http"}]}`,
			},
		},
		Status: corev1.PodStatus{
			Phase: "Running",
			PodIP: "192.168.1.1",
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 8080,
						},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Port: intstr.FromInt32(8081),
							},
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Port: intstr.FromInt32(8082),
							},
						},
					},
				},
			},
		},
	}
	if modifier != nil {
		modifier(&newPod)
	}
	return &newPod
}

func getTestClient(stopCh <-chan struct{}) (*client.Client, kubernetes.Interface) {
	clientset := testclient.NewSimpleClientset()
	k8sclient := client.CreateClient(clientset, stopCh)
	return k8sclient, clientset
}
