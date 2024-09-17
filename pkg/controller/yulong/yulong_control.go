package yulong

import (
	"context"
	"strconv"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
)

type ControlInterface interface {
	// UpdateYuLong implements the control logic for StatefulSet creation, update, and deletion
	UpdateYuLong(yl *v1alpha1.YuLong) error
}

func NewDefaultYuLongControl(
	deps *controller.Dependencies,
	conditionUpdater YuLongConditionUpdater,
	recorder record.EventRecorder) *defaultYuLongControl {
	return &defaultYuLongControl{
		deps:             deps,
		conditionUpdater: conditionUpdater,
		recorder:         recorder}
}

type defaultYuLongControl struct {
	deps             *controller.Dependencies
	conditionUpdater YuLongConditionUpdater
	recorder         record.EventRecorder
}

func (c defaultYuLongControl) UpdateYuLong(yl *v1alpha1.YuLong) error {
	var errs []error

	if err := c.updateYuLong(yl); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (c *defaultYuLongControl) updateYuLong(yl *v1alpha1.YuLong) error {
	ns := yl.GetNamespace()
	name := yl.GetName()

	tc, err := c.deps.TiDBClusterLister.TidbClusters(ns).Get(name)
	if err != nil && errors.IsNotFound(err) {
		return err
	}

	pdClient := controller.GetPDClient(c.deps.PDControl, tc)
	stores, err := pdClient.GetStores()
	if err != nil {
		return err
	}

	err = c.conditionUpdater.Update(yl, tc)
	if err != nil {
		return err
	}

	newStatus, err := c.deps.Clientset.PingcapV1alpha1().YuLongs(ns).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	newStatus.Status.UsedSize = strconv.Itoa(int(stores.Stores[0].Status.Capacity - stores.Stores[0].Status.Available))
	newStatus.Status.Conditions = yl.Status.Conditions

	_, err = c.deps.Clientset.PingcapV1alpha1().YuLongs(ns).Update(context.TODO(), newStatus, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}
