package autoregistration

import (
	"context"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/steadybit/extension-auto-registration-kubernetes/autoregistration"
	"github.com/steadybit/extension-auto-registration-kubernetes/client"
	"github.com/steadybit/extension-auto-registration-kubernetes/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type TestSupport struct {
	addPod           func(*corev1.Pod)
	deletePod        func(*corev1.Pod)
	updatePod        func(*corev1.Pod)
	addService       func(*corev1.Service)
	deleteService    func(*corev1.Service)
	updateService    func(*corev1.Service)
	getRegistrations func() (added []string, removed []string)
}

func TestAutoRegistration_should_register_extensions(t *testing.T) {
	type args struct {
		matchLabels        config.Labels
		matchLabelsExclude config.Labels
	}
	tests := []struct {
		name string
		args args
		test func(t *testing.T, ts TestSupport)
	}{
		{
			name: "should add daemonset pod without service",
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(nil))
				added, _ := ts.getRegistrations()
				assert.Len(t, added, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://192.168.1.1:8080\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\"},\"restrictedIps\":[\"192.168.1.1\"]}", added[0])
			},
		},
		{
			name: "should add deployment pod for existing service",
			test: func(t *testing.T, ts TestSupport) {
				ts.addService(getTestService(nil))
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				added, _ := ts.getRegistrations()
				assert.Len(t, added, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://test-service.default.svc.cluster.local:8085\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\",\"8085\":\"ServicePort\"},\"restrictedIps\":[\"555.555.555.555\",\"192.168.1.1\"]}", added[0])
			},
		},
		{
			name: "should add deployment pod when service is created after pod",
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				ts.addService(getTestService(nil))
				added, _ := ts.getRegistrations()
				assert.Len(t, added, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://test-service.default.svc.cluster.local:8085\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\",\"8085\":\"ServicePort\"},\"restrictedIps\":[\"555.555.555.555\",\"192.168.1.1\"]}", added[0])
			},
		},
		{
			name: "should remove registration when pod is deleted",
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(nil))
				added, _ := ts.getRegistrations()
				assert.Len(t, added, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://192.168.1.1:8080\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\"},\"restrictedIps\":[\"192.168.1.1\"]}", added[0])
				ts.deletePod(getTestPod(nil))
				added, removed := ts.getRegistrations()
				assert.Len(t, added, 1, "There should still only be one added extension.")
				assert.Len(t, removed, 1, "There should be one removed extension.")
				assert.Equal(t, "{\"url\":\"http://192.168.1.1:8080\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\"},\"restrictedIps\":[\"192.168.1.1\"]}", removed[0])
			},
		},
		{
			name: "should remove registration when daemonset pod is updated and annotations are removed",
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(nil))
				added, _ := ts.getRegistrations()
				assert.Len(t, added, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://192.168.1.1:8080\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\"},\"restrictedIps\":[\"192.168.1.1\"]}", added[0])
				ts.updatePod(getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				added, removed := ts.getRegistrations()
				assert.Len(t, added, 1, "There should still only be one added extension.")
				assert.Len(t, removed, 1, "There should be one removed extension.")
				assert.Equal(t, "{\"url\":\"http://192.168.1.1:8080\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\"},\"restrictedIps\":[\"192.168.1.1\"]}", removed[0])
			},
		},
		{
			name: "should add registration when daemonset pod is updated and annotations are added",
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				added, _ := ts.getRegistrations()
				assert.Empty(t, added, "Nothing should be registered")
				ts.updatePod(getTestPod(nil))
				added, removed := ts.getRegistrations()
				assert.Len(t, added, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://192.168.1.1:8080\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\"},\"restrictedIps\":[\"192.168.1.1\"]}", added[0])
				assert.Empty(t, removed, "Nothing should be removed")
			},
		},
		{
			name: "should remove registration when service is deleted",
			test: func(t *testing.T, ts TestSupport) {
				ts.addService(getTestService(nil))
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				added, _ := ts.getRegistrations()
				assert.Len(t, added, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://test-service.default.svc.cluster.local:8085\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\",\"8085\":\"ServicePort\"},\"restrictedIps\":[\"555.555.555.555\",\"192.168.1.1\"]}", added[0])
				ts.deleteService(getTestService(nil))
				added, removed := ts.getRegistrations()
				assert.Len(t, added, 1, "There should still only be one added extension.")
				assert.Len(t, removed, 1, "There should be one removed extension.")
				assert.Equal(t, "{\"url\":\"http://test-service.default.svc.cluster.local:8085\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\",\"8085\":\"ServicePort\"},\"restrictedIps\":[\"555.555.555.555\",\"192.168.1.1\"]}", removed[0])
			},
		},
		{
			name: "should remove registration when service is updated and annotations are removed",
			test: func(t *testing.T, ts TestSupport) {
				ts.addService(getTestService(nil))
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				added, _ := ts.getRegistrations()
				assert.Len(t, added, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://test-service.default.svc.cluster.local:8085\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\",\"8085\":\"ServicePort\"},\"restrictedIps\":[\"555.555.555.555\",\"192.168.1.1\"]}", added[0])
				ts.updateService(getTestService(func(p *corev1.Service) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				added, removed := ts.getRegistrations()
				assert.Len(t, added, 1, "There should still only be one added extension.")
				assert.Len(t, removed, 1, "There should be one removed extension.")
				assert.Equal(t, "{\"url\":\"http://test-service.default.svc.cluster.local:8085\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\",\"8085\":\"ServicePort\"},\"restrictedIps\":[\"555.555.555.555\",\"192.168.1.1\"]}", added[0])
			},
		},
		{
			name: "should add registration when service is updated and annotations are added",
			test: func(t *testing.T, ts TestSupport) {
				ts.addService(getTestService(func(p *corev1.Service) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				added, _ := ts.getRegistrations()
				assert.Empty(t, added, "Nothing should be registered")
				ts.updateService(getTestService(nil))
				added, _ = ts.getRegistrations()
				assert.Len(t, added, 1, "There should be one added extension.")
				assert.Equal(t, "{\"url\":\"http://test-service.default.svc.cluster.local:8085\",\"restrictedPorts\":{\"8080\":\"ContainerPort\",\"8081\":\"LivenessProbe\",\"8082\":\"ReadinessProbe\",\"8085\":\"ServicePort\"},\"restrictedIps\":[\"555.555.555.555\",\"192.168.1.1\"]}", added[0])
			},
		},
		{
			name: "should ignore pod without annotations",
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.ObjectMeta.Annotations = map[string]string{}
				}))
				added, _ := ts.getRegistrations()
				assert.Empty(t, added, "Nothing should be registered")
			},
		},
		{
			name: "should ignore pod without ip",
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.Status.PodIP = ""
				}))
				added, _ := ts.getRegistrations()
				assert.Empty(t, added, "Nothing should be registered")
			},
		},
		{
			name: "should ignore not running pod",
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.Status.Phase = "Pending"
				}))
				added, _ := ts.getRegistrations()
				assert.Empty(t, added, "Nothing should be registered")
			},
		},
		{
			name: "should ignore not ready pod",
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(func(p *corev1.Pod) {
					p.Status.Conditions = []corev1.PodCondition{}
				}))
				added, _ := ts.getRegistrations()
				assert.Empty(t, added, "Nothing should be registered")
			},
		},
		{
			name: "should ignore pod not matching matchLabels",
			args: args{
				matchLabels: config.Labels{
					{
						Key:   "app",
						Value: "extension-abc",
					},
				},
			},
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(nil))
				added, _ := ts.getRegistrations()
				assert.Empty(t, added, "Nothing should be registered")
			},
		},
		{
			name: "should ignore pod matching matchLabelsExclude",
			args: args{
				matchLabelsExclude: config.Labels{
					{
						Key:   "app",
						Value: "extension-xyz",
					},
				},
			},
			test: func(t *testing.T, ts TestSupport) {
				ts.addPod(getTestPod(nil))
				added, _ := ts.getRegistrations()
				assert.Empty(t, added, "Nothing should be registered")
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
			k8sclient, k8stestclient := getTestClient(stopCh)

			config.Config.AgentRegistrationInterval = 1 * time.Second
			config.Config.AgentRegistrationIntervalAfterError = 1 * time.Second
			config.Config.MatchLabels = tt.args.matchLabels
			config.Config.MatchLabelsExclude = tt.args.matchLabelsExclude
			registrator := autoregistration.UpdateAgentExtensions(httpClient, k8sclient)

			tt.test(t, TestSupport{
				addPod: func(pod *corev1.Pod) {
					_, err := k8stestclient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
					assert.NoError(t, err, "Pod creation should succeed")
					time.Sleep(100 * time.Millisecond)
					waitUntilSynched(t, registrator)
				},
				deletePod: func(pod *corev1.Pod) {
					err := k8stestclient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
					assert.NoError(t, err, "Pod deletion should succeed")
					time.Sleep(100 * time.Millisecond)
					waitUntilSynched(t, registrator)
				},
				updatePod: func(pod *corev1.Pod) {
					_, err := k8stestclient.CoreV1().Pods(pod.Namespace).Update(context.Background(), pod, metav1.UpdateOptions{})
					assert.NoError(t, err, "Pod update should succeed")
					time.Sleep(100 * time.Millisecond)
					waitUntilSynched(t, registrator)
				},
				addService: func(svc *corev1.Service) {
					_, err := k8stestclient.CoreV1().Services(svc.Namespace).Create(context.Background(), svc, metav1.CreateOptions{})
					assert.NoError(t, err, "Service creation should succeed")
					time.Sleep(100 * time.Millisecond)
					waitUntilSynched(t, registrator)
				},
				deleteService: func(svc *corev1.Service) {
					err := k8stestclient.CoreV1().Services(svc.Namespace).Delete(context.Background(), svc.Name, metav1.DeleteOptions{})
					assert.NoError(t, err, "Service deletion should succeed")
					time.Sleep(100 * time.Millisecond)
					waitUntilSynched(t, registrator)
				},
				updateService: func(svc *corev1.Service) {
					_, err := k8stestclient.CoreV1().Services(svc.Namespace).Update(context.Background(), svc, metav1.UpdateOptions{})
					assert.NoError(t, err, "Service update should succeed")
					time.Sleep(100 * time.Millisecond)
					waitUntilSynched(t, registrator)
				},
				getRegistrations: func() (added []string, removed []string) {
					MU.RLock()
					defer MU.RUnlock()
					added = AddedExtensions
					removed = RemovedExtensions
					return
				},
			})
		})
	}
}

func waitUntilSynched(t *testing.T, registrator *autoregistration.AutoRegistration) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !registrator.IsDirty() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("Registrator did not sync in time")
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

func getTestService(modifier func(p *corev1.Service)) *corev1.Service {
	newService := corev1.Service{
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
	}
	if modifier != nil {
		modifier(&newService)
	}
	return &newService
}

func getTestClient(stopCh <-chan struct{}) (*client.Client, kubernetes.Interface) {
	clientset := testclient.NewSimpleClientset()
	k8sclient := client.CreateClient(clientset, stopCh)
	return k8sclient, clientset
}
