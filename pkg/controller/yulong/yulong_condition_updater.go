package yulong

import (
	"fmt"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	utilyulong "github.com/pingcap/tidb-operator/pkg/util/yulong"
	v1 "k8s.io/api/core/v1"
)

// YuLongConditionUpdater interface that translates cluster state
// into YuLong status conditions.
type YuLongConditionUpdater interface {
	Update(*v1alpha1.YuLong, *v1alpha1.TidbCluster) error
}

type yuLongConditionUpdater struct {
}

var _ YuLongConditionUpdater = &yuLongConditionUpdater{}

func (u *yuLongConditionUpdater) Update(yl *v1alpha1.YuLong, tc *v1alpha1.TidbCluster) error {
	u.updateReadyCondition(tc, yl)
	// in the future, we may return error when we need to Kubernetes API, etc.
	return nil
}

func TiDBClusterAreUpToDate(tc *v1alpha1.TidbCluster) bool {
	return tc.Status.Conditions[0].Status == v1.ConditionTrue
}

func (u *yuLongConditionUpdater) updateReadyCondition(tc *v1alpha1.TidbCluster, yl *v1alpha1.YuLong) {
	status := v1.ConditionFalse
	reason := ""
	message := ""

	switch {
	case !TiDBClusterAreUpToDate(tc):
		reason = utilyulong.ClusterUnReady
		message = "TiDB cluster is not ready"
	default:
		status = v1.ConditionTrue
		reason = utilyulong.Ready
		message = "TiDB cluster is fully up and running"
	}
	cond := utilyulong.NewYuLongCondition(v1alpha1.YuLongReady, status, reason, message)
	fmt.Println("cond", cond)
	utilyulong.SetYuLongCondition(&yl.Status, *cond)
}
