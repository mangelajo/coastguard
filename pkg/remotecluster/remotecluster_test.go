package remotecluster

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v1net "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog"
)

const clusterID1 = "test-cluster-1"
const pod0 = "pod0"
const np0 = "np0"

var _ = Describe("Coastguard RemoteCluster", func() {
	klog.InitFlags(nil)

	var eventChannel chan *Event

	BeforeEach(func() {
		eventChannel = make(chan *Event, 10)
	})

	Context("Synchronization", func() {

		It("Should notify when cluster synchronization has finished", func() {
			remoteCluster := New(clusterID1, fake.NewSimpleClientset())
			defer remoteCluster.Stop()

			var done chan bool = make(chan bool)

			remoteCluster.Run(func(*RemoteCluster) {
				done <- true
			})
			// Wait for sync first
			Eventually(done).Should(Receive(BeTrue()))
		})

		It("Should eventually return HasSynced true", func() {
			remoteCluster := New(clusterID1, fake.NewSimpleClientset())
			defer remoteCluster.Stop()

			remoteCluster.Run(nil)

			Eventually(remoteCluster.HasSynced).Should(BeTrue())
		})
	})

	Context("Event handling", func() {
		It("Should send events on discovered pods", func() {

			remoteCluster, _ := createRemoteClusterWithPod(eventChannel)
			defer remoteCluster.Stop()

			By("Waiting for the Pod AddEvent to be received")
			var event *Event
			Eventually(eventChannel).Should(Receive(&event))
			Expect(event.Type).Should(Equal(AddEvent))
			Expect(event.ObjType).Should(Equal(Pod))

			Consistently(eventChannel).ShouldNot(Receive())

		})

		It("Should send events on discovered NetworkPolicies", func() {

			remoteCluster, _ := createRemoteClusterWithNetworkPolicy(eventChannel)
			defer remoteCluster.Stop()

			By("Waiting for the NetworkPolicy AddEvent to be received")
			var event *Event
			Eventually(eventChannel).Should(Receive(&event))
			Expect(event.Type).Should(Equal(AddEvent))
			Expect(event.ObjType).Should(Equal(NetworkPolicy))

			Consistently(eventChannel).ShouldNot(Receive())

		})

		It("Should discover newly created Pods", func() {
			remoteCluster := createRemoteClusterWithObjects(eventChannel)

			_, err := remoteCluster.ClientSet.CoreV1().Pods("default").Create(NewPod("pod1"))
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for the Pod AddEvent to be received")
			var event *Event
			Eventually(eventChannel).Should(Receive(&event))
			Expect(event.Type).Should(Equal(AddEvent))
			Expect(event.ObjType).Should(Equal(Pod))
			Expect(event.Objs).Should(HaveLen(1))

		})

		It("Should discover newly created NetworkPolicies", func() {
			remoteCluster := createRemoteClusterWithObjects(eventChannel)

			_, err := remoteCluster.ClientSet.NetworkingV1().NetworkPolicies("default").Create(NewNetworkPolicy(np0))
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for the NetworkPolicy AddEvent to be received")
			var event *Event
			Eventually(eventChannel).Should(Receive(&event))
			Expect(event.Type).Should(Equal(AddEvent))
			Expect(event.ObjType).Should(Equal(NetworkPolicy))
			Expect(event.Objs).Should(HaveLen(1))

		})

		It("Should discover Pods being deleted", func() {
			remoteCluster, _ := createRemoteClusterWithPod(eventChannel)
			defer remoteCluster.Stop()
			cs := remoteCluster.ClientSet

			By("Deleting the test pod")
			err := cs.CoreV1().Pods("default").Delete(pod0, &metav1.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for the Pod AddEvent, then DeleteEvent to be received")

			for _, rcvType := range []EventType{AddEvent, DeleteEvent} {
				var event *Event
				Eventually(eventChannel).Should(Receive(&event))
				Expect(event.Type).Should(Equal(rcvType))
				Expect(event.Objs).Should(HaveLen(1))
			}
		})

		It("Should discover NetworkPolicies being deleted", func() {
			remoteCluster, _ := createRemoteClusterWithNetworkPolicy(eventChannel)
			defer remoteCluster.Stop()
			cs := remoteCluster.ClientSet

			By("Deleting the test pod")
			err := cs.NetworkingV1().NetworkPolicies("default").Delete(np0, &metav1.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for the NetworkPolicy AddEvent, then DeleteEvent to be received")

			for _, rcvType := range []EventType{AddEvent, DeleteEvent} {
				var event *Event
				Eventually(eventChannel).Should(Receive(&event))
				Expect(event.Type).Should(Equal(rcvType))
				Expect(event.Objs).Should(HaveLen(1))
			}
		})

		It("It should discover pods being updated", func() {
			remoteCluster, pod := createRemoteClusterWithPod(eventChannel)
			defer remoteCluster.Stop()
			cs := remoteCluster.ClientSet

			By("Updating the test pod label")
			pod.SetLabels(map[string]string{"label-one": "1"})
			_, err := cs.CoreV1().Pods("default").Update(pod)
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for the Pod AddEvent, then UpdateEvent to be received")

			Eventually(eventChannel).Should(Receive())
			var event *Event
			Eventually(eventChannel).Should(Receive(&event))
			Expect(event.Type).Should(Equal(UpdateEvent))
			Expect(event.Objs).Should(HaveLen(2))

		})

		It("It should discover NetworkPolicies being updated", func() {
			remoteCluster, np := createRemoteClusterWithNetworkPolicy(eventChannel)
			defer remoteCluster.Stop()
			cs := remoteCluster.ClientSet

			By("Updating the test NetworkPolicy label")
			np.SetLabels(map[string]string{"label-one": "1"})
			_, err := cs.NetworkingV1().NetworkPolicies("default").Update(np)
			Expect(err).ShouldNot(HaveOccurred())

			By("Waiting for the NetworkPolicy AddEvent, then UpdateEvent to be received")

			Eventually(eventChannel).Should(Receive())
			var event *Event
			Eventually(eventChannel).Should(Receive(&event))
			Expect(event.Type).Should(Equal(UpdateEvent))
			Expect(event.Objs).Should(HaveLen(2))

		})
	})

	Context("Finalization of the RemoteCluster watcher", func() {
		It("Should return stopped once we stop it", func() {
			remoteCluster, _ := createRemoteClusterWithPod(eventChannel)
			Expect(remoteCluster.Stopped()).To(BeFalse())
			remoteCluster.Stop()
			Expect(remoteCluster.Stopped()).To(BeTrue())
		})
	})

	Context("Access to the informers cache", func() {
		It("Should be able to list existing pods in informer cache", func() {
			remoteCluster, _ := createRemoteClusterWithPod(eventChannel)
			Expect(remoteCluster.GetPods()).Should(HaveLen(1))
		})
	})
})

func createRemoteClusterWithObjects(eventChannel chan *Event, objects ...runtime.Object) *RemoteCluster {
	clientSet := fake.NewSimpleClientset(objects...)
	By("Creating a new remoteCluster with a clientset of one pod")
	remoteCluster := New(clusterID1, clientSet)
	remoteCluster.SetEventChannel(eventChannel)

	var done chan bool = make(chan bool)

	remoteCluster.Run(func(*RemoteCluster) {
		done <- true
	})
	// Wait for sync first
	Eventually(done).Should(Receive(BeTrue()))
	return remoteCluster
}

func createRemoteClusterWithPod(eventChannel chan *Event) (*RemoteCluster, *v1.Pod) {
	testPod := NewPod(pod0)

	return createRemoteClusterWithObjects(eventChannel,
		&v1.PodList{Items: []v1.Pod{*testPod}}), testPod

}

func createRemoteClusterWithNetworkPolicy(eventChannel chan *Event) (*RemoteCluster, *v1net.NetworkPolicy) {
	testNetworkPolicy := NewNetworkPolicy(np0)

	return createRemoteClusterWithObjects(eventChannel,
		&v1net.NetworkPolicyList{Items: []v1net.NetworkPolicy{*testNetworkPolicy}}), testNetworkPolicy

}

func NewPod(name string) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
		},
		Status: v1.PodStatus{
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
	}

	return pod
}

func NewNetworkPolicy(name string) *v1net.NetworkPolicy {
	np := &v1net.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      name,
		},
	}
	return np
}

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coastguard: RemoteCluster suite")
}
