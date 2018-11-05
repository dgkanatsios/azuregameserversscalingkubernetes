package main

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type clusterState struct {
	runningDGSCount               int32
	totalPodCount                 int32
	failedDGSNotInCollectionCount int32
	failedDGSInCollectionCount    int32
	markedForDeletionDGSCount     int32
	dgsColState                   dgsv1alpha1.DedicatedGameServerCollectionState
	podColState                   corev1.PodPhase
}

func validateClusterState(state clusterState) {
	log.Infof("    Waiting for %d seconds", delayInSeconds)
	time.Sleep(time.Duration(delayInSeconds) * time.Second)

	if state.dgsColState == "" {
		state.dgsColState = dgsv1alpha1.DGSColRunning
	}

	if state.podColState == "" {
		state.podColState = corev1.PodRunning
	}

	log.Infof("    Verifying that %d pods are in state %s", state.totalPodCount, corev1.PodRunning)
	err := loopCheck(verifyPods, state.totalPodCount)
	if err != nil {
		handleError(err)
	}

	log.Infof("    Verifying that %d DedicatedGameServers are Running", state.runningDGSCount)
	err = loopCheck(verifyRunningDedicatedGameServers, state.runningDGSCount)
	if err != nil {
		handleError(err)
	}

	log.Infof("    Verifying that DedicatedGameServerCollection with %d replicas is in state DGS: %s, Pod: %s", state.runningDGSCount, state.dgsColState, state.podColState)
	err = loopCheckDGSCol(verifyDedicatedGameServerCollection, state.runningDGSCount, state.dgsColState, state.podColState)
	if err != nil {
		handleError(err)
	}

	log.Infof("    Verifying that %d DedicatedGameServers are Failed and do not belong in the DedicatedGameServerCollection", state.failedDGSNotInCollectionCount)
	err = loopCheck(verifyFailedDedicatedGameServersNotInCollection, state.failedDGSNotInCollectionCount)
	if err != nil {
		handleError(err)
	}

	log.Infof("    Verifying that %d DedicatedGameServers are Failed and belong in the DedicatedGameServerCollection", state.failedDGSInCollectionCount)
	err = loopCheck(verifyFailedDedicatedGameServersInCollection, state.failedDGSInCollectionCount)
	if err != nil {
		handleError(err)
	}

	log.Infof("    Verifying that we have %d DGS with state MarkedForDeletion", state.markedForDeletionDGSCount)
	err = loopCheck(verifyMarkedForDeletionDedicatedGameServers, state.markedForDeletionDGSCount)
	if err != nil {
		handleError(err)
	}
}

func verifyPods(count int32) error {
	pods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: shared.LabelIsDedicatedGameServer,
	})
	if err != nil {
		log.Error("Cannot get pods")
		return err
	}

	dgsPods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			if lbl, ok := pod.ObjectMeta.Labels[shared.LabelDedicatedGameServerName]; ok {
				if strings.HasPrefix(lbl, dgsColName) {
					if pod.ObjectMeta.Labels[shared.LabelIsDedicatedGameServer] == "true" {
						if len(pod.OwnerReferences) > 0 && strings.HasPrefix(pod.OwnerReferences[0].Name, dgsColName) {
							dgsPods++
						}
					}
				}
			}
		}
	}
	if dgsPods == int(count) {
		return nil
	}
	return errors.New("Pods not OK")
}

func verifyRunningDedicatedGameServers(count int32) error {
	dgss, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Error("Cannot get DGSs")
		return err
	}

	dgsCount := 0
	for _, dgs := range dgss.Items {
		if dgs.Status.PodState == corev1.PodRunning && dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSRunning {
			if lbl, ok := dgs.ObjectMeta.Labels[shared.LabelDedicatedGameServerCollectionName]; ok {
				if lbl == dgsColName {
					if len(dgs.OwnerReferences) > 0 && dgs.OwnerReferences[0].Name == dgsColName {
						dgsCount++
					}
				}
			}
		}
	}
	if dgsCount == int(count) {
		return nil
	}
	return errors.New("DGSs not OK")
}

func verifyFailedDedicatedGameServersNotInCollection(count int32) error {
	dgss, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Error("Cannot get DGSs")
		return err
	}

	dgsCount := 0
	for _, dgs := range dgss.Items {
		if dgs.Status.PodState == corev1.PodRunning && dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSFailed {
			if lbl, ok := dgs.ObjectMeta.Labels[shared.LabelOriginalDedicatedGameServerCollectionName]; ok {
				if lbl == dgsColName {
					if len(dgs.OwnerReferences) == 0 {
						dgsCount++
					}
				}
			}
		}
	}
	if dgsCount == int(count) {
		return nil
	}
	return errors.New("DGSs not OK")
}

func verifyFailedDedicatedGameServersInCollection(count int32) error {
	dgss, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Error("Cannot get DGSs")
		return err
	}

	dgsCount := 0
	for _, dgs := range dgss.Items {
		if dgs.Status.PodState == corev1.PodRunning && dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSFailed {
			if lbl, ok := dgs.ObjectMeta.Labels[shared.LabelDedicatedGameServerCollectionName]; ok {
				if lbl == dgsColName {
					if len(dgs.OwnerReferences) > 0 && dgs.OwnerReferences[0].Name == dgsColName {
						dgsCount++
					}
				}
			}
		}
	}
	if dgsCount == int(count) {
		return nil
	}
	return errors.New("DGSs not OK")
}

func verifyDedicatedGameServerCollection(availableReplicas int32, dgsColState dgsv1alpha1.DedicatedGameServerCollectionState, podColState corev1.PodPhase) error {
	dgsCol, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Get(dgsColName, metav1.GetOptions{})
	if err != nil {
		log.Error("Cannot get DGSCol")
		return err
	}
	if dgsCol.Status.AvailableReplicas == availableReplicas &&
		dgsCol.Status.DedicatedGameServerCollectionState == dgsColState &&
		dgsCol.Status.PodCollectionState == podColState {
		return nil
	}
	return fmt.Errorf("Could not get %d available replicas", availableReplicas)
}

func verifyMarkedForDeletionDedicatedGameServers(count int32) error {
	dgss, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServers(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Error("Cannot get DGSs")
		return err
	}

	dgsCount := 0
	for _, dgs := range dgss.Items {
		if dgs.Status.PodState == corev1.PodRunning && dgs.Status.DedicatedGameServerState == dgsv1alpha1.DGSMarkedForDeletion {
			if lbl, ok := dgs.ObjectMeta.Labels[shared.LabelOriginalDedicatedGameServerCollectionName]; ok {
				if lbl == dgsColName {
					if len(dgs.OwnerReferences) == 0 {
						dgsCount++
					}
				}
			}
		}
	}
	if dgsCount == int(count) {
		return nil
	}
	return errors.New("MarkedForDeletion DGSs not OK")
}

func verifyDGSTimesFailed(times int32) error {
	dgsCol, err := dgsclient.AzuregamingV1alpha1().DedicatedGameServerCollections(namespace).Get(dgsColName, metav1.GetOptions{})
	if err != nil {
		log.Error("Cannot get DGSCol")
		return err
	}
	if dgsCol.Status.DGSTimesFailed != times {
		return fmt.Errorf("Incorrect DGSTimesFailed. It is %d whereas it should be %d", dgsCol.Status.DGSTimesFailed, times)
	}
	return nil
}

func loopCheck(fn func(int32) error, count int32) error {
	var err error
	for times := 0; times < loopTimes; times++ {
		err = fn(count)
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(delayInSecondsForLoopTest) * time.Second)
	}
	return fmt.Errorf("Could not get %d available object types, error: %s", count, err.Error())
}

func loopCheckDGSCol(fn func(int32, dgsv1alpha1.DedicatedGameServerCollectionState, corev1.PodPhase) error,
	count int32, dgsColState dgsv1alpha1.DedicatedGameServerCollectionState, podColState corev1.PodPhase) error {
	var err error
	for times := 0; times < loopTimes; times++ {
		err = fn(count, dgsColState, podColState)
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(delayInSecondsForLoopTest) * time.Second)
	}
	return fmt.Errorf("Could not get %d available object types, error: %s", count, err.Error())
}

func getRandomInt(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
