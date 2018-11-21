package dgs

import (
	"fmt"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	logrus "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
)

// hasDGSChanged returns true if *all* of the following DGS properties have changed
// dgsHealth, podPhase, publicIP, nodeName, activePlayers
// As expected, it returns false if at least one has changed
func (c *DGSController) hasDGSChanged(oldDGS, newDGS *dgsv1alpha1.DedicatedGameServer) bool {

	//check if any new containers have been added
	if len(oldDGS.Spec.Template.Containers) != len(newDGS.Spec.Template.Containers) {
		return true
	}

	//check if any of the images has changed
	for i := 0; i < len(oldDGS.Spec.Template.Containers); i++ {
		if oldDGS.Spec.Template.Containers[i].Image != newDGS.Spec.Template.Containers[i].Image {
			return true
		}
	}

	// we check if all of the following fields are the same
	if oldDGS.Status.Health != newDGS.Status.Health ||
		oldDGS.Status.PodPhase != newDGS.Status.PodPhase ||
		oldDGS.Status.PublicIP != newDGS.Status.PublicIP ||
		oldDGS.Status.NodeName != newDGS.Status.NodeName ||
		oldDGS.Status.ActivePlayers != newDGS.Status.ActivePlayers ||
		!shared.AreMapsSame(oldDGS.Labels, newDGS.Labels) {

		return true
	}

	return false
}

func (c *DGSController) handleDGSMarkedForDeletionWithZeroPlayers(dgsTemp *dgsv1alpha1.DedicatedGameServer) error {
	err := c.dgsClient.AzuregamingV1alpha1().DedicatedGameServers(dgsTemp.Namespace).Delete(dgsTemp.Name, &metav1.DeleteOptions{})
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"Name":  dgsTemp.Name,
			"Error": err.Error(),
		}).Error("Cannot delete DedicatedGameServer")
		runtime.HandleError(fmt.Errorf("DedicatedGameServer '%s' cannot be deleted", dgsTemp.Name))
		return err
	}
	c.logger.WithField("Name", dgsTemp.Name).Info("DedicatedGameServer with state MarkedForDeletion and with 0 ActivePlayers was deleted")
	c.recorder.Event(dgsTemp, corev1.EventTypeNormal, shared.SuccessSynced, fmt.Sprintf(shared.MessageMarkedForDeletionDedicatedGameServerDeleted, dgsTemp.Name))
	return nil //nothing more to do here
}

func (c *DGSController) getPublicIPForNode(nodeName string) (string, error) {
	node, err := c.nodeLister.Get(nodeName)
	if err != nil {
		return "", err
	}
	for _, x := range node.Status.Addresses {
		if x.Type == corev1.NodeExternalIP {
			return x.Address, nil
		}
	}
	c.logger.Infof("Node with name %s does not have a Public IP, will try to return the InternalIP", nodeName)
	// externalIP not found, try InternalIP
	for _, x := range node.Status.Addresses {
		if x.Type == corev1.NodeInternalIP {
			return x.Address, nil
		}
	}
	return "", fmt.Errorf("Node with name %s does not have a Public or Internal IP", nodeName)
}

func (c *DGSController) isDGSMarkedForDeletionWithZeroPlayers(dgs *dgsv1alpha1.DedicatedGameServer) bool {
	//check its state and active players
	return dgs.Status.ActivePlayers == 0 && dgs.Status.MarkedForDeletion
}

func (c *DGSController) isDGSFailed(dgs *dgsv1alpha1.DedicatedGameServer) bool {
	return dgs.Status.Health == dgsv1alpha1.DGSFailed
}
