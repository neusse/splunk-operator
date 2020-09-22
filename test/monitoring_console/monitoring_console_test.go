package monitoringconsoletest

import (
	"os/exec"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	enterprisev1 "github.com/splunk/splunk-operator/pkg/apis/enterprise/v1alpha3"
	splcommon "github.com/splunk/splunk-operator/pkg/splunk/common"
	"github.com/splunk/splunk-operator/test/testenv"
)

var _ = Describe("Smoke test", func() {

	var deployment *testenv.Deployment

	BeforeEach(func() {
		var err error
		deployment, err = testenvInstance.NewDeployment(testenv.RandomDNSName(3))
		Expect(err).To(Succeed(), "Unable to create deployment")
	})

	AfterEach(func() {
		// When a test spec failed, skip the teardown so we can troubleshoot.
		if CurrentGinkgoTestDescription().Failed {
			testenvInstance.SkipTeardown = true
		}
		if deployment != nil {
			deployment.Teardown()
		}
		if !testenvInstance.SkipTeardown {
			testenv.DeleteMCPod(testenvInstance.GetName())
		}
	})

	Context("Standalone deployment (S1)", func() {
		It("can deploy a MC with standalone instance and update MC with new standalone deployment", func() {

			standaloneOneName := deployment.GetName()
			standaloneOne, err := deployment.DeployStandalone(standaloneOneName)
			Expect(err).To(Succeed(), "Unable to deploy standalone instance ")

			Eventually(func() splcommon.Phase {
				err = deployment.GetInstance(standaloneOneName, standaloneOne)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for standalone instance status to be ready", "instance", standaloneOne.ObjectMeta.Name, "Phase", standaloneOne.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())

				return standaloneOne.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseReady))

			// In a steady state, we should stay in Ready and not flip-flop around
			Consistently(func() splcommon.Phase {
				_ = deployment.GetInstance(deployment.GetName(), standaloneOne)
				return standaloneOne.Status.Phase
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(splcommon.PhaseReady))

			// Wait for 1 MC pod in namespace
			Eventually(func() int {
				count := testenv.GetMCPodCount(testenvInstance.GetName())
				return count
			}, deployment.GetTimeout(), PollInterval).Should(Equal(1))

			// Monitoring Console Pod is in Ready State
			Eventually(func() bool {
				check := testenv.CheckMCPodReady(testenvInstance.GetName())
				return check
			}, deployment.GetTimeout(), PollInterval).Should(Equal(true))

			// Check Monitoring console is configured with all standalone instances in namespace
			peerList := testenv.GetConfiguredPeers(testenvInstance.GetName())
			testenvInstance.Log.Info("Peer List", "instance", peerList)

			// Only 1 peer expected in MC peer list
			Expect(len(peerList)).To(Equal(1))

			podName := "splunk-" + standaloneOneName + "-standalone-0"
			testenvInstance.Log.Info("Check standalone instance in MC Peer list", "Standalone Pod", podName, "Peer in peer list", peerList[0])
			Expect(strings.Contains(peerList[0], podName)).To(Equal(true))

			// Add another standalone instance in namespace
			standaloneTwoName := deployment.GetName() + "-two"
			standaloneTwo, err := deployment.DeployStandalone(standaloneTwoName)
			Expect(err).To(Succeed(), "Unable to deploy standalone instance ")

			Eventually(func() splcommon.Phase {
				err = deployment.GetInstance(standaloneTwoName, standaloneTwo)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for standalone instance status to be ready", "instance", standaloneTwo.ObjectMeta.Name, "Phase", standaloneTwo.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())

				return standaloneTwo.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseReady))

			// In a steady state, we should stay in Ready and not flip-flop around
			Consistently(func() splcommon.Phase {
				_ = deployment.GetInstance(standaloneTwoName, standaloneTwo)
				return standaloneTwo.Status.Phase
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(splcommon.PhaseReady))

			// Wait for new MC Pod to come up and existing MC Pod to terminate
			Eventually(func() int {
				count := testenv.GetMCPodCount(testenvInstance.GetName())
				return count
			}, deployment.GetTimeout(), PollInterval).Should(Equal(1))

			// Monitoring Console Pod is in Ready State
			Eventually(func() bool {
				check := testenv.CheckMCPodReady(testenvInstance.GetName())
				return check
			}, deployment.GetTimeout(), PollInterval).Should(Equal(true))

			// Check Monitoring console is configured with all standalone instances in namespace
			peerList = testenv.GetConfiguredPeers(testenvInstance.GetName())
			configuredStandaloneOne := false
			confguredStandaloneTwo := false
			testenvInstance.Log.Info("Peer List", "instance", peerList)

			// Only 2 peer expected in MC peer list
			Expect(len(peerList)).To(Equal(2))

			podNameOne := "splunk-" + standaloneOneName + "-standalone-0"
			podNameTwo := "splunk-" + standaloneTwoName + "-standalone-0"
			for _, peer := range peerList {
				if strings.Contains(peer, podNameOne) {
					testenvInstance.Log.Info("Check standalone instance in MC Peer list", "Standalone Pod", podNameOne, "Peer in peer list", peer)
					configuredStandaloneOne = true
					continue
				}
				if strings.Contains(peer, podNameTwo) {
					confguredStandaloneTwo = true
					testenvInstance.Log.Info("Check standalone instance in MC Peer list", "Standalone Pod", podNameTwo, "Peer in peer list", peer)
				}
			}
			Expect(configuredStandaloneOne && confguredStandaloneTwo).To(Equal(true))
		})
	})

	Context("Standalone deployment with Scale up", func() {
		It("can deploy a MC with standalone instance and update MC when standalone is scaled up", func() {

			standalone, err := deployment.DeployStandalone(deployment.GetName())
			Expect(err).To(Succeed(), "Unable to deploy standalone instance ")

			Eventually(func() splcommon.Phase {
				err = deployment.GetInstance(deployment.GetName(), standalone)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for standalone instance status to be ready", "instance", standalone.ObjectMeta.Name, "Phase", standalone.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())

				return standalone.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseReady))

			// In a steady state, we should stay in Ready and not flip-flop around
			Consistently(func() splcommon.Phase {
				_ = deployment.GetInstance(deployment.GetName(), standalone)
				return standalone.Status.Phase
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(splcommon.PhaseReady))

			// Wait for 1 MC pod in namespace
			Eventually(func() int {
				count := testenv.GetMCPodCount(testenvInstance.GetName())
				return count
			}, deployment.GetTimeout(), PollInterval).Should(Equal(1))

			// Monitoring Console Pod is in Ready State
			Eventually(func() bool {
				check := testenv.CheckMCPodReady(testenvInstance.GetName())
				return check
			}, deployment.GetTimeout(), PollInterval).Should(Equal(true))

			// Check Monitoring console is configured with all standalone instances in namespace
			peerList := testenv.GetConfiguredPeers(testenvInstance.GetName())
			testenvInstance.Log.Info("Peer List", "instance", peerList)

			// Only 1 peer expected in MC peer list
			Expect(len(peerList)).To(Equal(1))

			// Check spluk standlone pods are configured in MC peer list
			podName := "splunk-" + deployment.GetName() + "-standalone-0"
			testenvInstance.Log.Info("Check standalone instance in MC Peer list", "Standalone Pod", podName, "Peer in peer list", peerList[0])
			Expect(strings.Contains(peerList[0], podName)).To(Equal(true))

			// Scale Standalone instance
			testenvInstance.Log.Info("Scaling standalone cluster")
			_, err = exec.Command("kubectl", "scale", "standalone", "-n", testenvInstance.GetName(), deployment.GetName(), "--replicas=2").Output()
			Expect(err).To(Succeed(), "Failed to execute scale up command")

			// Ensure standalone is scaling up
			Eventually(func() splcommon.Phase {
				err := deployment.GetInstance(deployment.GetName(), standalone)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for standalone status to be Scaling Up", "instance", standalone.ObjectMeta.Name, "Phase", standalone.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())
				return standalone.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseScalingUp))

			// Ensure Standalone go to Ready phase
			Eventually(func() splcommon.Phase {
				err := deployment.GetInstance(deployment.GetName(), standalone)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for standalone status to be READY", "instance", standalone.ObjectMeta.Name, "Phase", standalone.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())
				return standalone.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseReady))

			// In a steady state, we should stay in Ready and not flip-flop around
			Consistently(func() splcommon.Phase {
				_ = deployment.GetInstance(deployment.GetName(), standalone)
				return standalone.Status.Phase
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(splcommon.PhaseReady))

			// Wait for new MC Pod to come up and existing MC Pod to terminate
			Eventually(func() int {
				count := testenv.GetMCPodCount(testenvInstance.GetName())
				testenvInstance.Log.Info("Waiting for montioring console pods count", "count", count)
				testenv.DumpGetPods(testenvInstance.GetName())
				return count
			}, deployment.GetTimeout(), PollInterval).Should(Equal(1))

			// Monitoring Console Pod is in Ready State
			Eventually(func() bool {
				check := testenv.CheckMCPodReady(testenvInstance.GetName())
				return check
			}, deployment.GetTimeout(), PollInterval).Should(Equal(true))

			// Only 2 peer expected in MC peer list
			peerList = testenv.GetConfiguredPeers(testenvInstance.GetName())
			testenvInstance.Log.Info("Peers in configuredPeer List", "count", len(peerList))
			Expect(len(peerList)).To(Equal(2))

			// Check standalone pods are configured  in MC Peer List
			found := make(map[string]bool)
			testenvInstance.Log.Info("Peer List", "instance", peerList)
			for i := 0; i < 2; i++ {
				podName := "splunk-" + deployment.GetName() + "-standalone-" + strconv.Itoa(i)
				found[podName] = false
				for _, peer := range peerList {
					if strings.Contains(peer, podName) {
						testenvInstance.Log.Info("Check Peer matches standalone pod", "Standalone Pod", podName, "Peer in peer list", peer)
						found[podName] = true
						break
					}
				}
			}
			allStandaloneConfigured := true
			for _, key := range found {
				if !key {
					allStandaloneConfigured = false
					break
				}
			}
			Expect(allStandaloneConfigured).To(Equal(true))
		})
	})

	Context("SearchHeadCluster deployment with Scale Up", func() {
		It("MC can configure SHC instances in a namespace", func() {

			_, err := deployment.DeploySearchHeadCluster(deployment.GetName(), "", "", "")
			Expect(err).To(Succeed(), "Unable to deploy search head cluster")
			// Ensure search head cluster go to Ready phase
			shc := &enterprisev1.SearchHeadCluster{}
			Eventually(func() splcommon.Phase {
				err := deployment.GetInstance(deployment.GetName(), shc)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for search head cluster instance status to be ready", "instance", shc.ObjectMeta.Name, "Phase", shc.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())
				return shc.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseReady))

			// In a steady state, we should stay in Ready and not flip-flop around
			Consistently(func() splcommon.Phase {
				_ = deployment.GetInstance(deployment.GetName(), shc)
				return shc.Status.Phase
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(splcommon.PhaseReady))

			// Wait for 1 MC pod in namespace
			Eventually(func() int {
				count := testenv.GetMCPodCount(testenvInstance.GetName())
				return count
			}, deployment.GetTimeout(), PollInterval).Should(Equal(1))

			// Check Monitoring Console Pod is in Ready State
			Eventually(func() bool {
				check := testenv.CheckMCPodReady(testenvInstance.GetName())
				return check
			}, deployment.GetTimeout(), PollInterval).Should(Equal(true))

			// Check Monitoring console is configured with all search head instances in namespace
			peerList := testenv.GetConfiguredPeers(testenvInstance.GetName())
			found := make(map[string]bool)
			testenvInstance.Log.Info("Peer List", "instance", peerList)
			for i := 0; i < 3; i++ {
				podName := "splunk-" + deployment.GetName() + "-search-head-" + strconv.Itoa(i)
				found[podName] = false
				for _, peer := range peerList {
					if strings.Contains(peer, podName) {
						testenvInstance.Log.Info("Check Peer matches serach head pod", "Search Head Pod", podName, "Peer in peer list", peer)
						found[podName] = true
						break
					}
				}
			}
			allSearchHeadsConfigured := true
			for _, key := range found {
				if !key {
					allSearchHeadsConfigured = false
					break
				}
			}
			Expect(allSearchHeadsConfigured).To(Equal(true))

			// Scale Search Head Cluster
			testenvInstance.Log.Info("Scaling search head cluster")
			_, err = exec.Command("kubectl", "scale", "shc", "-n", testenvInstance.GetName(), deployment.GetName(), "--replicas=4").Output()
			Expect(err).To(Succeed(), "Failed to scale search head cluster")

			// Ensure search head cluster go to ScalingUp phase
			Eventually(func() splcommon.Phase {
				err := deployment.GetInstance(deployment.GetName(), shc)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for search head cluster instance status to be Scaling Up", "instance", shc.ObjectMeta.Name, "Phase", shc.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())
				return shc.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseScalingUp))

			// Ensure search head cluster go to Ready phase
			Eventually(func() splcommon.Phase {
				err := deployment.GetInstance(deployment.GetName(), shc)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for search head cluster instance status to be READY", "instance", shc.ObjectMeta.Name, "Phase", shc.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())
				return shc.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseReady))

			// In a steady state, we should stay in Ready and not flip-flop around
			Consistently(func() splcommon.Phase {
				_ = deployment.GetInstance(deployment.GetName(), shc)
				return shc.Status.Phase
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(splcommon.PhaseReady))

			// Check New MC Pod comes up and old one is terminated
			Eventually(func() int {
				mcPodCount := testenv.GetMCPodCount(testenvInstance.GetName())
				return mcPodCount
			}, deployment.GetTimeout(), PollInterval).Should(Equal(1))

			// Monitoring Console Pod is in Ready State
			Eventually(func() bool {
				check := testenv.CheckMCPodReady(testenvInstance.GetName())
				return check
			}, deployment.GetTimeout(), PollInterval).Should(Equal(true))

			// Check Monitoring console is configured with all search head instances in namespace
			peerList = testenv.GetConfiguredPeers(testenvInstance.GetName())
			found = make(map[string]bool)
			testenvInstance.Log.Info("Peer List", "instance", peerList)
			for i := 0; i < 4; i++ {
				podName := "splunk-" + deployment.GetName() + "-search-head-" + strconv.Itoa(i)
				found[podName] = false
				for _, peer := range peerList {
					if strings.Contains(peer, podName) {
						testenvInstance.Log.Info("Check Peer matches serach head pod", "Search Head Pod", podName, "Peer in peer list", peer)
						found[podName] = true
						break
					}
				}
			}
			allSearchHeadsConfigured = true
			for _, key := range found {
				if !key {
					allSearchHeadsConfigured = false
					break
				}
			}
			Expect(allSearchHeadsConfigured).To(Equal(true))
		})
	})

	Context("SearchHeadCluster and Standalone", func() {
		It("MC can configure SHC and Standalone instances in a namespace", func() {

			_, err := deployment.DeploySearchHeadCluster(deployment.GetName(), "", "", "")
			Expect(err).To(Succeed(), "Unable to deploy search head cluster")
			// Ensure search head cluster go to Ready phase
			shc := &enterprisev1.SearchHeadCluster{}
			Eventually(func() splcommon.Phase {
				err := deployment.GetInstance(deployment.GetName(), shc)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for search head cluster instance status to be ready", "instance", shc.ObjectMeta.Name, "Phase", shc.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())
				return shc.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseReady))

			// In a steady state, we should stay in Ready and not flip-flop around
			Consistently(func() splcommon.Phase {
				_ = deployment.GetInstance(deployment.GetName(), shc)
				return shc.Status.Phase
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(splcommon.PhaseReady))

			// Wait for 1 MC pod in namespace
			Eventually(func() int {
				count := testenv.GetMCPodCount(testenvInstance.GetName())
				return count
			}, deployment.GetTimeout(), PollInterval).Should(Equal(1))

			// Monitoring Console Pod is in Ready State
			Eventually(func() bool {
				check := testenv.CheckMCPodReady(testenvInstance.GetName())
				return check
			}, deployment.GetTimeout(), PollInterval).Should(Equal(true))

			// Check Monitoring console is configured with all search head instances in namespace
			peerList := testenv.GetConfiguredPeers(testenvInstance.GetName())
			found := make(map[string]bool)
			testenvInstance.Log.Info("Peer List", "instance", peerList)
			for i := 0; i < 3; i++ {
				podName := "splunk-" + deployment.GetName() + "-search-head-" + strconv.Itoa(i)
				found[podName] = false
				for _, peer := range peerList {
					if strings.Contains(peer, podName) {
						testenvInstance.Log.Info("Check Peer matches serach head pod", "Search Head Pod", podName, "Peer in peer list", peer)
						found[podName] = true
						break
					}
				}
			}
			allSearchHeadsConfigured := true
			for _, key := range found {
				if !key {
					allSearchHeadsConfigured = false
					break
				}
			}
			Expect(allSearchHeadsConfigured).To(Equal(true))

			// Deploy Standalone
			standalone, err := deployment.DeployStandalone(deployment.GetName())
			Expect(err).To(Succeed(), "Unable to deploy standalone instance ")

			Eventually(func() splcommon.Phase {
				err = deployment.GetInstance(deployment.GetName(), standalone)
				if err != nil {
					return splcommon.PhaseError
				}
				testenvInstance.Log.Info("Waiting for standalone instance status to be ready", "instance", standalone.ObjectMeta.Name, "Phase", standalone.Status.Phase)
				testenv.DumpGetPods(testenvInstance.GetName())

				return standalone.Status.Phase
			}, deployment.GetTimeout(), PollInterval).Should(Equal(splcommon.PhaseReady))

			// In a steady state, we should stay in Ready and not flip-flop around
			Consistently(func() splcommon.Phase {
				_ = deployment.GetInstance(deployment.GetName(), standalone)
				return standalone.Status.Phase
			}, ConsistentDuration, ConsistentPollInterval).Should(Equal(splcommon.PhaseReady))

			// Check New MC Pod comes up and old one is terminated
			Eventually(func() int {
				mcPodCount := testenv.GetMCPodCount(testenvInstance.GetName())
				return mcPodCount
			}, deployment.GetTimeout(), PollInterval).Should(Equal(1))

			// Monitoring Console Pod is in Ready State
			Eventually(func() bool {
				check := testenv.CheckMCPodReady(testenvInstance.GetName())
				return check
			}, deployment.GetTimeout(), PollInterval).Should(Equal(true))

			// Get Peers configured on Monitoring Console
			peerList = testenv.GetConfiguredPeers(testenvInstance.GetName())
			found = make(map[string]bool)
			testenvInstance.Log.Info("Peer List", "instance", peerList)

			// Check for SearchHead Peers in Peer List
			for i := 0; i < 3; i++ {
				podName := "splunk-" + deployment.GetName() + "-search-head-" + strconv.Itoa(i)
				found[podName] = false
				for _, peer := range peerList {
					if strings.Contains(peer, podName) {
						testenvInstance.Log.Info("Check Peer matches search head pod", "Search Head Pod", podName, "Peer in peer list", peer)
						found[podName] = true
						break
					}
				}
			}

			// Check Standalone configured on Monitoring Console
			podName := "splunk-" + deployment.GetName() + "-standalone-0"
			found[podName] = false
			for _, peer := range peerList {
				if strings.Contains(peer, podName) {
					testenvInstance.Log.Info("Check Peer matches Standalone pod", "Standalone Pod", podName, "Peer in peer list", peer)
					found[podName] = true
					break
				}
			}

			// Verify all instances are configured on Monitoring Console
			allInstancesConfigured := true
			for _, key := range found {
				if !key {
					allInstancesConfigured = false
					break
				}
			}
			Expect(allInstancesConfigured).To(Equal(true))
		})
	})
})
