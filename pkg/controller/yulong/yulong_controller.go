package yulong

import (
	"fmt"
	"time"

	perrors "github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	"github.com/pingcap/tidb-operator/pkg/metrics"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller controls yulong.
type Controller struct {
	deps *controller.Dependencies
	// control returns an interface capable of syncing a yulong.
	// Abstracted out for testing.
	control ControlInterface
	// yulong that need to be synced.
	queue workqueue.RateLimitingInterface
}

func NewController(deps *controller.Dependencies) *Controller {
	c := &Controller{
		deps: deps,
		control: NewDefaultYuLongControl(
			deps,
			&yuLongConditionUpdater{},
			deps.Recorder,
		),
		queue: workqueue.NewNamedRateLimitingQueue(
			controller.NewControllerRateLimiter(1*time.Second, 100*time.Second),
			"YuLong",
		),
	}
	YuLongInformer := deps.InformerFactory.Pingcap().V1alpha1().YuLongs()
	YuLongInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueueYuLong,
		UpdateFunc: func(old, cur interface{}) {
			c.enqueueYuLong(cur)
		},
		DeleteFunc: c.enqueueYuLong,
	})
	return c
}

// Name returns the name of the YuLong controller
func (c *Controller) Name() string {
	return "YuLong"
}

// Run runs the tidbcluster controller.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	klog.Info("Starting YuLong controller")
	defer klog.Info("Shutting down YuLong controller")

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}

	<-stopCh
}

// worker runs a worker goroutine that invokes processNextWorkItem until the the controller's queue is closed
func (c *Controller) worker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem dequeues items, processes them, and marks them done. It enforces that the syncHandler is never
// invoked concurrently with the same key.
func (c *Controller) processNextWorkItem() bool {
	metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(1)
	defer metrics.ActiveWorkers.WithLabelValues(c.Name()).Add(-1)

	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)
	if err := c.sync(key.(string)); err != nil {
		if perrors.Find(err, controller.IsRequeueError) != nil {
			klog.Infof("YuLong: %v, still need sync: %v, requeuing", key.(string), err)
		} else {
			utilruntime.HandleError(fmt.Errorf("YuLong: %v, sync failed %v, requeuing", key.(string), err))
		}
		c.queue.AddRateLimited(key)
	} else {
		c.queue.Forget(key)
	}
	return true
}

// sync syncs the given YuLong.
func (c *Controller) sync(key string) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		metrics.ReconcileTime.WithLabelValues(c.Name()).Observe(duration.Seconds())
		klog.V(4).Infof("Finished syncing YuLong %q (%v)", key, duration)
	}()

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	yt, err := c.deps.YuLongLister.YuLongs(ns).Get(name)
	if err != nil {
		return err
	}

	if errors.IsNotFound(err) {
		klog.Infof("YuLong has been deleted %v", key)
		return nil
	}

	return c.syncYuLong(yt.DeepCopy())
}

func (c *Controller) syncYuLong(yl *v1alpha1.YuLong) error {
	return c.control.UpdateYuLong(yl.DeepCopy())
}

// enqueueTidbCluster enqueues the given YuLong in the work queue.
func (c *Controller) enqueueYuLong(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Cound't get key for object %+v: %v", obj, err))
		return
	}
	c.queue.Add(key)
}
