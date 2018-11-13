package main

import (
	"fmt"
	"time"

	"k8s.io/client-go/util/retry"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsclientsetversioned "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var client *kubernetes.Clientset
var dgsclient *dgsclientsetversioned.Clientset

const (
	dgsColName                string = "simplenodejsudp"
	namespace                 string = "default"
	loopTimes                 int    = 20
	delayInSecondsForLoopTest int    = 5
	delayInSeconds            int    = 1
)

func main() {

	var err error
	client, dgsclient, err = shared.GetClientSet()
	if err != nil {
		log.Errorf("Cannot initialize connection to cluster due to: %v", err)
	}

	// on this step, we do check if there are 5 running DGSs
	log.Info("Step 1")
	initialValidation() // check if there are 5 DGS

	// we incrase the replicas to 10, so we should have 10 DGS and 10 Pods
	// we also set ActivePlayers for all DGS to 2
	log.Info("Step 2a")
	// scale out
	increaseReplicasToTen()                                                    // kubectl scale dgsc ... --replicas=10
	validateClusterState(clusterState{totalPodCount: 10, runningDGSCount: 10}) // check if there are 10 DGS and 10 pods
	setAllActivePlayers(2)                                                     // set all ActivePlayers to two

	// decrease from 10 to 7
	// so our DGSCol should now contain 7 DGS, whereas 3 DGS should be out of the collection in MarkedForDeletion state
	log.Info("Step 2b")
	decreaseReplicasToSeven() // kubectl scale dgsc ... --replicas=7
	validateClusterState(clusterState{totalPodCount: 10, runningDGSCount: 7, markedForDeletionDGSCount: 3})

	// make players leave from the MarkedForDeletionServers
	// these DGSs should be deleted
	log.Info("Step 3")
	// make players leave
	setActivePlayersOfMarkedForDeletionToZero() // set all ActivePlayers to zero
	validateClusterState(clusterState{totalPodCount: 7, runningDGSCount: 7, markedForDeletionDGSCount: 0})

	// introduce some chaos doing random pod/DGS deletes
	log.Info("Step 4a")
	// some random deletes
	deleteRandomDGS() // delete a random DGS
	deleteRandomPod() // delete a random DGS-ownered pod
	deleteRandomDGS() // delete a random DGS
	validateClusterState(clusterState{totalPodCount: 7, runningDGSCount: 7})

	// more chaos
	log.Info("Step 4b")
	deleteRandomPod() // delete a random DGS-ownered pod
	deleteRandomDGS() // delete a random DGS
	deleteRandomPod() // delete a random DGS-ownered pod
	validateClusterState(clusterState{totalPodCount: 7, runningDGSCount: 7})

	// let's test some DGS failure - recall that DGSCol.FailBehavior is set to Remove. DGSMaxFailures is 2
	log.Info("Step 5a")
	setRandomRunningDGSToFailed() // set a random's DGS GameState to Failed, this DGS should be removed from the collection
	// runningDGSCount should be 7 because another one will be created in the Failed one's place
	validateClusterState(clusterState{
		totalPodCount:                 8,
		runningDGSCount:               7,
		failedDGSInCollectionCount:    0,
		failedDGSNotInCollectionCount: 1,
	})
	validateDGSTimesFailed(1) // check that DGS times failed is 1

	// let's fail another one
	// we're about to surpass the threshold of 2
	log.Info("Step 5b")
	setRandomRunningDGSToFailed() // set a random's DGS GameState to Failed, this DGS should be removed from the collection
	validateClusterState(clusterState{
		totalPodCount:                 9,
		runningDGSCount:               7,
		failedDGSInCollectionCount:    0,
		failedDGSNotInCollectionCount: 2,
	})
	//runningDGSCount is 7 because another one will be created
	validateDGSTimesFailed(2) // check that DGS times failed is 2

	// another failure
	// as we will surpass the MaxFailures threshold, this one failed DGS will not be removed from the collection
	// and the DGSCOlState is NeedsIntervention
	// no change in the number of Pods
	log.Info("Step 5c")
	setRandomRunningDGSToFailed()
	validateClusterState(clusterState{
		totalPodCount:                 9,
		runningDGSCount:               6,
		failedDGSInCollectionCount:    1,
		failedDGSNotInCollectionCount: 2,
		dgsColState:                   dgsv1alpha1.DGSColNeedsIntervention,
	})
	validateDGSTimesFailed(2)

	// another one
	// no change, since we are in the NeedsIntervention state
	// only one more DGS failed
	log.Info("Step 5d")
	setRandomRunningDGSToFailed()
	validateClusterState(clusterState{
		totalPodCount:                 9,
		runningDGSCount:               5,
		failedDGSInCollectionCount:    2,
		failedDGSNotInCollectionCount: 2,
		dgsColState:                   dgsv1alpha1.DGSColNeedsIntervention,
	})
	validateDGSTimesFailed(2)

	// reset the cluster, delete all DGS, set replicas to 5
	// reset DGSTimesFailed to zero
	// set the DGSFailBehavior to Delete
	log.Info("Step 6a")
	resetClusterState(5, dgsv1alpha1.Delete)
	validateClusterState(clusterState{
		totalPodCount:                 5,
		runningDGSCount:               5,
		failedDGSInCollectionCount:    0,
		failedDGSNotInCollectionCount: 0,
		dgsColState:                   dgsv1alpha1.DGSColRunning,
	})
	validateDGSTimesFailed(0)

	// a random failure
	// DGSTimesFailed will become 1
	log.Info("Step 6b")
	setRandomRunningDGSToFailed()
	validateClusterState(clusterState{
		totalPodCount:                 5,
		runningDGSCount:               5,
		failedDGSInCollectionCount:    0,
		failedDGSNotInCollectionCount: 0,
		dgsColState:                   dgsv1alpha1.DGSColRunning,
	})
	validateDGSTimesFailed(1)

	// another random failure
	// DGSTimesFailed will become 2
	log.Info("Step 6c")
	setRandomRunningDGSToFailed()
	validateClusterState(clusterState{
		totalPodCount:                 5,
		runningDGSCount:               5,
		failedDGSInCollectionCount:    0,
		failedDGSNotInCollectionCount: 0,
		dgsColState:                   dgsv1alpha1.DGSColRunning,
	})
	validateDGSTimesFailed(2)

	// another random failure
	// we surpassed the threshold
	// DGSTimesFailed will stay 2
	// state will become NeedsIntervention
	log.Info("Step 6d")
	setRandomRunningDGSToFailed()
	validateClusterState(clusterState{
		totalPodCount:                 5,
		runningDGSCount:               4,
		failedDGSInCollectionCount:    1,
		failedDGSNotInCollectionCount: 0,
		dgsColState:                   dgsv1alpha1.DGSColNeedsIntervention,
	})
	validateDGSTimesFailed(2)

	// another random failure
	// state is NeedsIntervention
	// DGSTimesFailed will stay 2
	log.Info("Step 6e")
	setRandomRunningDGSToFailed()
	validateClusterState(clusterState{
		totalPodCount:                 5,
		runningDGSCount:               3,
		failedDGSInCollectionCount:    2,
		failedDGSNotInCollectionCount: 0,
		dgsColState:                   dgsv1alpha1.DGSColNeedsIntervention,
	})
	validateDGSTimesFailed(2)

	// reset the cluster, delete all DGS, set replicas to 5
	// reset DGSTimesFailed to zero
	// set the DGSFailBehavior to Delete
	// add DGS autoscaler - minimum Replicas 5, maximum 7
	log.Info("Step 7a")
	resetClusterState(5, dgsv1alpha1.Remove)
	// HACK - on resetClusterState, DGSs are delete while the DGSCol is in NeedsIntervention - so no more DGS will be created
	// so we delete a randomDGS after it in order to trigger the controller
	deleteRandomDGS()
	validateClusterState(clusterState{
		totalPodCount:   5,
		runningDGSCount: 5,
		dgsColState:     dgsv1alpha1.DGSColRunning,
	})
	addAutoScalerDetails()
	setAllActivePlayers(9)
	// verify that autoscaler has kicked in and we have one more DGS
	validateClusterState(clusterState{
		totalPodCount:   6,
		runningDGSCount: 6,
		dgsColState:     dgsv1alpha1.DGSColRunning,
	})

	// set again 9 players for all DGS - 1 new DGS will be created
	log.Info("Step 7b")
	setAutoscalerLastScaleOperationDateTimeToZeroValue()
	setAllActivePlayers(9)
	// verify that autoscaler has kicked in and we have one more DGS
	validateClusterState(clusterState{
		totalPodCount:   7,
		runningDGSCount: 7,
		dgsColState:     dgsv1alpha1.DGSColRunning,
	})

	// set again 9 players for all DGS - no new DGS will be created since we are at the maximum of 7
	log.Info("Step 7c")
	setAutoscalerLastScaleOperationDateTimeToZeroValue()
	setAllActivePlayers(9)
	validateClusterState(clusterState{
		totalPodCount:   7,
		runningDGSCount: 7,
		dgsColState:     dgsv1alpha1.DGSColRunning,
	})

	// set 5 players for all DGS -> 1 DGS less
	log.Info("Step 7d")
	setAutoscalerLastScaleOperationDateTimeToZeroValue()
	setAllActivePlayers(5)
	validateClusterState(clusterState{
		totalPodCount:             7, // 7 pods: 6 in collection, 1 out
		runningDGSCount:           6,
		dgsColState:               dgsv1alpha1.DGSColRunning,
		markedForDeletionDGSCount: 1,
	})

	// set again 5 players for all DGS -> 1 DGS less
	log.Info("Step 7e")
	setAutoscalerLastScaleOperationDateTimeToZeroValue()
	setAllActivePlayers(5)
	validateClusterState(clusterState{
		totalPodCount:             7, // 7 pods: 5 in collection, 2 out
		runningDGSCount:           5,
		dgsColState:               dgsv1alpha1.DGSColRunning,
		markedForDeletionDGSCount: 2,
	})

	// set 5 players for all DGS -> not going less than the minimum (5) replicas
	log.Info("Step 7f")
	setAutoscalerLastScaleOperationDateTimeToZeroValue()
	setAllActivePlayers(5)
	validateClusterState(clusterState{
		totalPodCount:             7,
		runningDGSCount:           5,
		dgsColState:               dgsv1alpha1.DGSColRunning,
		markedForDeletionDGSCount: 2,
	})

}

func handleError(err error) {
	log.Panic(err)
}

func setAutoscalerLastScaleOperationDateTimeToZeroValue() {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dgscol, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Get(dgsColName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		dgscol.Spec.DgsAutoScalerDetails.LastScaleOperationDateTime = ""
		_, err = dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgscol)
		if err != nil {
			return err
		}
		return nil
	})
	if retryErr != nil {
		handleError(retryErr)
	}
}

func validateDGSTimesFailed(times int32) {
	log.Infof("    Verifying that DGSTimesFailed is %d", times)
	err := loopCheck(verifyDGSTimesFailed, times)
	if err != nil {
		handleError(err)
	}
}

func addAutoScalerDetails() {
	log.Info("    Adding DGS AutoScaler")

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dgscol, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Get(dgsColName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		dgscol.Spec.DgsAutoScalerDetails = &dgsv1alpha1.DedicatedGameServerDgsAutoScalerDetails{
			MinimumReplicas:     5,
			MaximumReplicas:     7,
			ScaleInThreshold:    60,
			ScaleOutThreshold:   80,
			Enabled:             true,
			CoolDownInMinutes:   5,
			MaxPlayersPerServer: 10,
		}
		_, err = dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgscol)
		return err
	})
	if retryErr != nil {
		handleError(fmt.Errorf("Cannot update DGSCol because of %s", retryErr.Error()))
	}
}

func resetClusterState(replicas int32, failbehavior dgsv1alpha1.DedicatedGameServerFailBehavior) {
	log.Info("    Resetting cluster state")
	dgss, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{})
	if err != nil {
		handleError(fmt.Errorf("Cannot get DGSs"))
	}

	//delete all of the failed ones
	for _, dgs := range dgss.Items {
		if dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSFailed {
			err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Delete(dgs.Name, &metav1.DeleteOptions{})
			if err != nil {
				handleError(fmt.Errorf("Cannot delete DGS because of %s", err))
			}
		}
	}

	log.Infof("    Waiting for %d seconds", delayInSeconds)
	time.Sleep(time.Duration(delayInSeconds) * time.Second)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dgscol, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Get(dgsColName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		dgscol.Spec.DGSFailBehavior = failbehavior
		dgscol.Spec.Replicas = replicas
		dgscol.Status.DedicatedGameServerCollectionState = dgsv1alpha1.DGSColCreating
		dgscol.Status.DGSTimesFailed = 0
		_, err = dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgscol)
		return err
	})
	if retryErr != nil {
		handleError(fmt.Errorf("Cannot update DGSCol because of %s", retryErr.Error()))
	}
}

func setRandomRunningDGSToFailed() {
	log.Info("    Setting a random running DGS from the collection to Failed")
	dgss, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{
		LabelSelector: shared.LabelDedicatedGameServerCollectionName,
	})
	if err != nil {
		handleError(fmt.Errorf("Cannot get DGSs"))
	}

	var dgsUpdate *dgsv1alpha1.DedicatedGameServer
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var dgs *dgsv1alpha1.DedicatedGameServer
		for {
			dgs = &dgss.Items[getRandomInt(0, len(dgss.Items)-1)]
			dgs, err = dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Get(dgs.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSRunning { // if it's Running
				dgs = dgs.DeepCopy()
				dgs.Status.DedicatedGameServerState = dgsv1alpha1.DGSFailed
				break
			}
		}
		dgsUpdate, err = dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Update(dgs)
		return err
	})
	if retryErr != nil {
		handleError(fmt.Errorf("Cannot update DGS"))
	}

	if dgsUpdate.Status.DedicatedGameServerState != dgsv1alpha1.DGSFailed {
		handleError(fmt.Errorf("DGS was not updated"))
	}
}

func deleteRandomPod() {
	log.Info("    Deleting a random pod")
	pods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		handleError(fmt.Errorf("Cannot get pods"))
	}
	err = client.CoreV1().Pods(namespace).Delete(pods.Items[getRandomInt(0, len(pods.Items)-1)].Name, &metav1.DeleteOptions{})
	if err != nil {
		handleError(fmt.Errorf("Cannot delete pod"))
	}
}

func deleteRandomDGS() {
	log.Info("    Deleting a random DGS from the collection")
	dgss, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{
		LabelSelector: shared.LabelDedicatedGameServerCollectionName,
	})
	if err != nil {
		handleError(fmt.Errorf("Cannot get DGSs"))
	}

	//delete a random element
	err = dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Delete(dgss.Items[getRandomInt(0, len(dgss.Items)-1)].Name, &metav1.DeleteOptions{})
	if err != nil {
		handleError(fmt.Errorf("Cannot delete DGS"))
	}
}

func setActivePlayersOfMarkedForDeletionToZero() {
	log.Info("    Setting active players of MarkedForDeletion DGS to zero")
	dgss, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{})
	if err != nil {
		handleError(fmt.Errorf("Cannot get DGSs"))
	}

	for _, dgs := range dgss.Items {
		if dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSMarkedForDeletion &&
			dgs.Status.ActivePlayers == 2 {
			dgsCopy := dgs.DeepCopy()
			dgsCopy.Status.ActivePlayers = 0
			var dgsUpdated *dgsv1alpha1.DedicatedGameServer
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				dgsUpdated, err = dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Update(dgsCopy)
				return err
			})
			if retryErr != nil {
				handleError(fmt.Errorf("Cannot update DGS %s", dgsCopy.Name))
			}
			if dgsUpdated.Status.ActivePlayers != 0 {
				handleError(fmt.Errorf("DGS %s was not updated", dgsCopy.Name))
			}
		}
	}
}

func setAllActivePlayers(playerscount int) {
	log.Infof("    Setting active players to %d for all DGS", playerscount)
	dgss, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{})
	if err != nil {
		handleError(fmt.Errorf("Cannot get DGSs"))
	}

	for _, dgs := range dgss.Items {
		var dgsUpdated *dgsv1alpha1.DedicatedGameServer
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			dgsCopy, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Get(dgs.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			dgsCopy.Status.ActivePlayers = playerscount
			dgsUpdated, err = dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).Update(dgsCopy)
			return err
		})
		if retryErr != nil {
			handleError(fmt.Errorf("Cannot update DGS %s because of %s", dgs.Name, err.Error()))
		}
		if dgsUpdated.Status.ActivePlayers != playerscount {
			handleError(fmt.Errorf("DGS %s was not updated with 2 Players, value is %d", dgs.Name, dgsUpdated.Status.ActivePlayers))
		}
	}
}

func increaseReplicasToTen() {
	log.Info("    Increasing replicas of DGSCol to 10")
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dgsCol, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Get(dgsColName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		dgsCol.Spec.Replicas = 10
		_, err = dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgsCol)
		return err
	})
	if retryErr != nil {
		handleError(fmt.Errorf("Cannot update DGSCol"))
	}
}

func decreaseReplicasToSeven() {
	log.Info("    Decreasing replicas of DGSCol to 7")
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dgsCol, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Get(dgsColName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		dgsCol.Spec.Replicas = 7
		_, err = dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Update(dgsCol)
		return err
	})
	if retryErr != nil {
		handleError(fmt.Errorf("Cannot update DGSCol"))
	}
}

func initialValidation() {
	count := int32(5)
	log.Infof("    Waiting for %d seconds", delayInSeconds)
	time.Sleep(time.Duration(delayInSeconds) * time.Second)

	log.Infof("    Verifying that %d pods are Running", count)
	err := loopCheck(verifyPods, count)
	if err != nil {
		log.Error(err)
	}

	log.Infof("    Verifying that %d DedicatedGameServers are Running", count)
	err = loopCheck(verifyRunningDedicatedGameServers, count)
	if err != nil {
		log.Error(err)
	}

	log.Infof("    Verifying that DedicatedGameServerCollection with %d replicas is Running", count)
	err = loopCheckDGSCol(verifyDedicatedGameServerCollection, count, dgsv1alpha1.DGSColRunning, corev1.PodRunning)
	if err != nil {
		log.Error(err)
	}
}
