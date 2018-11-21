package dgscollection

import (
	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	logrus "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
)

func (c *DGSCollectionController) hasSpecChanged(oldDGSCol, newDGSCol *dgsv1alpha1.DedicatedGameServerCollection) bool {
	return oldDGSCol.Spec.Replicas != newDGSCol.Spec.Replicas
}

func (c *DGSCollectionController) setPodCollectionState(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) error {
	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsCol.Name,
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsInstances, err := c.dgsLister.DedicatedGameServers(dgsCol.Namespace).List(selector)

	if err != nil {
		return err
	}

	for _, dgs := range dgsInstances {
		//one pod is not running
		if dgs.Status.PodPhase != corev1.PodRunning {
			// so set the collection's Pod State to this one Pod's value
			dgsCol.Status.PodCollectionState = dgs.Status.PodPhase
			if dgsCol.Status.PodCollectionState == "" {
				dgsCol.Status.PodCollectionState = corev1.PodPending
			}
			return nil
		}
	}
	//all pods are running, so set the collection Pod State with the running value
	dgsCol.Status.PodCollectionState = corev1.PodRunning
	return nil

}

func (c *DGSCollectionController) setDedicatedGameServerCollectionHealth(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) error {
	if dgsCol.Status.DGSCollectionHealth == dgsv1alpha1.DGSColNeedsIntervention {
		return nil
	}

	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsCol.Name,
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsInstances, err := c.dgsLister.DedicatedGameServers(dgsCol.Namespace).List(selector)

	if err != nil {
		return err
	}

	for _, dgs := range dgsInstances {
		//at least one of the DGS is not running
		if dgs.Status.Health != dgsv1alpha1.DGSHealthy {
			//so set the overall collection state as the state of this one
			dgsCol.Status.DGSCollectionHealth = dgsv1alpha1.DGSColHealth(dgs.Status.Health)
			return nil
		}
	}
	//all of the DGS are running, so set the DGSCol state as running
	dgsCol.Status.DGSCollectionHealth = dgsv1alpha1.DGSColHealthy
	return nil
}

func (c *DGSCollectionController) setAvailableReplicasStatus(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) error {
	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsCol.Name,
	}
	// we search via Labels, each DGS will have the DGSCol name as a Label
	selector := labels.SelectorFromSet(set)
	dgsInstances, err := c.dgsLister.DedicatedGameServers(dgsCol.Namespace).List(selector)

	if err != nil {
		return err
	}

	dgsCol.Status.AvailableReplicas = 0

	for _, dgs := range dgsInstances {
		if dgs.Status.Health == dgsv1alpha1.DGSHealthy && dgs.Status.PodPhase == corev1.PodRunning {
			dgsCol.Status.AvailableReplicas++
		}
	}

	return nil
}

func (c *DGSCollectionController) addDGSColReplicas(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, dgsExistingCount int) error {
	//create them
	increaseCount := int(dgsCol.Spec.Replicas) - dgsExistingCount
	c.logger.WithFields(logrus.Fields{"DGSColName": dgsCol.Name, "IncreaseCount": increaseCount}).Printf("Scaling out")

	for i := 0; i < increaseCount; i++ {
		dgs := shared.NewDedicatedGameServer(dgsCol, dgsCol.Spec.Template)
		// if we want to expose ports for this DGS
		if dgsCol.Spec.PortsToExpose != nil {
			// for each container on the pod
			for k := 0; k < len(dgs.Spec.Template.Containers); k++ {
				// assign random port for each port request
				for j := 0; j < len(dgs.Spec.Template.Containers[k].Ports); j++ {
					// if we want to expose this specific ContainerPort
					if shared.SliceContains(dgsCol.Spec.PortsToExpose, dgs.Spec.Template.Containers[k].Ports[j].ContainerPort) {
						hostport, errPort := c.portRegistry.GetNewPort() //get a random port
						if errPort != nil {
							return errPort
						}
						dgs.Spec.Template.Containers[k].Ports[j].HostPort = hostport
					}
				}
			}
		}

		_, err := c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(dgsCol.Namespace).Create(dgs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *DGSCollectionController) removeDGSColReplicas(dgsColTemp *dgsv1alpha1.DedicatedGameServerCollection, dgsExisting []*dgsv1alpha1.DedicatedGameServer) error {
	dgsExistingCount := len(dgsExisting)
	// we need to decrease our DGS for this collection
	// to accomplish this, we'll first find the number of DGS we need to decrease
	decreaseCount := dgsExistingCount - int(dgsColTemp.Spec.Replicas)
	// we'll remove random instances of DGS from our DGSCol
	indexesToDecrease := shared.GetRandomIndexes(dgsExistingCount, decreaseCount)

	c.logger.WithFields(logrus.Fields{"DGSColName": dgsColTemp.Name, "DecreaseCount": decreaseCount}).Printf("Scaling in")

	for i := 0; i < len(indexesToDecrease); i++ {
		dgsToMarkForDeletionTemp, err := c.dgsLister.DedicatedGameServers(dgsColTemp.Namespace).Get(dgsExisting[indexesToDecrease[i]].Name)

		if err != nil {
			return err
		}
		dgsToMarkForDeletionToUpdate := dgsToMarkForDeletionTemp.DeepCopy()
		// update the DGS so it has no owners
		dgsToMarkForDeletionToUpdate.ObjectMeta.OwnerReferences = nil
		//remove the DGSCol name from the DGS labels
		delete(dgsToMarkForDeletionToUpdate.ObjectMeta.Labels, shared.LabelDedicatedGameServerCollectionName)
		//set its state as marked for deletion
		dgsToMarkForDeletionToUpdate.Status.MarkedForDeletion = true
		//set its previous Collection owner
		dgsToMarkForDeletionToUpdate.ObjectMeta.Labels[shared.LabelOriginalDedicatedGameServerCollectionName] = dgsColTemp.Name
		//update the DGS CRD
		_, err = c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(dgsColTemp.Namespace).Update(dgsToMarkForDeletionToUpdate)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *DGSCollectionController) increaseTimesFailed(dgsCol *dgsv1alpha1.DedicatedGameServerCollection, count int) {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dgsColToUpdate, err := c.dgsClient.AzuregamingV1alpha1().DedicatedGameServerCollections(dgsCol.Namespace).Get(dgsCol.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		dgsColToUpdate.Status.DGSTimesFailed += int32(count)

		_, err = c.dgsClient.AzuregamingV1alpha1().DedicatedGameServerCollections(dgsCol.Namespace).Update(dgsColToUpdate)
		if err == nil {
			c.logger.WithFields(logrus.Fields{"DedicatedGameServerCollection": dgsCol.Name}).Infof("Increased DGSTimesFailed field of the DGSCol with value %d, new value is %d", count, dgsColToUpdate.Status.DGSTimesFailed)
		}
		return err
	})
	if retryErr != nil {
		c.logger.WithFields(logrus.Fields{"DedicatedGameServerCollection": dgsCol.Name, "Error": retryErr.Error()}).Errorf("Failed to update DGSTimesFailed field of the DGSCol because of %s", retryErr.Error())
	}
}

func (c *DGSCollectionController) getNotFailedDGSForDGSCol(dgsColTemp *dgsv1alpha1.DedicatedGameServerCollection) ([]*dgsv1alpha1.DedicatedGameServer, error) {
	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsColTemp.Name,
	}

	// we do not want the failed ones
	selector := labels.SelectorFromSet(set)
	dgss, err := c.dgsLister.DedicatedGameServers(dgsColTemp.Namespace).List(selector)

	dgsToReturn := make([]*dgsv1alpha1.DedicatedGameServer, 0)

	for _, dgs := range dgss {
		if dgs.Status.Health != dgsv1alpha1.DGSFailed {
			dgsToReturn = append(dgsToReturn, dgs)
		}
	}

	return dgsToReturn, err
}

func (c *DGSCollectionController) getFailedDGSForDGSCol(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) ([]*dgsv1alpha1.DedicatedGameServer, error) {
	set := labels.Set{
		shared.LabelDedicatedGameServerCollectionName: dgsCol.Name,
	}

	// we do not want the failed ones
	selector := labels.SelectorFromSet(set)
	dgss, err := c.dgsLister.DedicatedGameServers(dgsCol.Namespace).List(selector)

	dgsToReturn := make([]*dgsv1alpha1.DedicatedGameServer, 0)

	for _, dgs := range dgss {
		if dgs.Status.Health == dgsv1alpha1.DGSFailed {
			dgsToReturn = append(dgsToReturn, dgs)
		}
	}

	return dgsToReturn, err
}

func (c *DGSCollectionController) setDGSColToNeedsIntervention(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dgsColToUpdate, err := c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(dgsCol.Namespace).Get(dgsCol.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		dgsColToUpdate.Status.DGSCollectionHealth = dgsv1alpha1.DGSColNeedsIntervention
		_, err = c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(dgsCol.Namespace).Update(dgsColToUpdate)

		if err != nil {
			return err
		}
		return nil
	})
	return retryErr
}

func (c *DGSCollectionController) reconcileStatuses(dgsCol *dgsv1alpha1.DedicatedGameServerCollection) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dgsColToUpdate, err := c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(dgsCol.Namespace).Get(dgsCol.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		err = c.setAvailableReplicasStatus(dgsColToUpdate)
		if err != nil {
			return err
		}

		//assign DGSCol.Status.DGSHealth
		err = c.setDedicatedGameServerCollectionHealth(dgsColToUpdate)
		if err != nil {
			return err
		}

		//assign DGSCol.Status.PodPhase
		err = c.setPodCollectionState(dgsColToUpdate)
		if err != nil {
			return err
		}

		_, err = c.dgsColClient.AzuregamingV1alpha1().DedicatedGameServerCollections(dgsCol.Namespace).Update(dgsColToUpdate)

		if err != nil {
			return err
		}
		return nil
	})
	return retryErr
}

func (c *DGSCollectionController) hasDGSStatusChanged(oldDGS, newDGS *dgsv1alpha1.DedicatedGameServer) bool {
	if oldDGS.Status.Health != newDGS.Status.Health ||
		oldDGS.Status.PodPhase != newDGS.Status.PodPhase ||
		len(oldDGS.GetOwnerReferences()) != len(newDGS.GetOwnerReferences()) {
		return true
	}
	return false
}
