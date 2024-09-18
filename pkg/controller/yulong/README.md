# Demo crd for YuLong

该分支在tidb operator的框架之下，开发了一个CRD，实时查看TiKV的used size，仅供入门学习参考。

![运行效果如下：](/pkg/controller/yulong/crd_yulong.png)


# 开发过程

## step 1

自定义CRD的type，并运行make generate生成改CRD的client, informer, crd.yaml, api docs等资源。

type.go 新增内容：
```golang
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// YuLong is the control script's spec
//
// +k8s:openapi-gen=true
// +kubebuilder:resource:shortName="yl"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="ClusterName",type=string,JSONPath=`.spec.clusterName`,description="The name of the tidb"
// +kubebuilder:printcolumn:name="UsedSize",type=string,JSONPath=`.status.usedSize`,description="The used size of the tiKV"
type YuLong struct {
        metav1.TypeMeta `json:",inline"`
        // +k8s:openapi-gen=false
        metav1.ObjectMeta `json:"metadata"`

        Spec YuLongSpec `json:"spec"`

        // +k8s:openapi-gen=false
        // Most recently observed status of the YuLong
        Status YuLongStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// YuLongList is YuLong list
// +k8s:openapi-gen=true
type YuLongList struct {
        metav1.TypeMeta `json:",inline"`
        // +k8s:openapi-gen=false
        metav1.ListMeta `json:"metadata"`

        Items []YuLong `json:"items"`
}

// YuLongSpec describes the attributes that a user creates on a tidb cluster
// +k8s:openapi-gen=true
type YuLongSpec struct {
        ClusterName string `json:"clusterName"`
}

type YuLongStatus struct {
        UsedSize string `json:"usedSize"`

        // Represents the latest available observations of a YuLong's state.
        // +optional
        // +nullable
        Conditions []YuLongCondition `json:"conditions,omitempty"`
}

// YuLongCondition describes the state of a yu long at a certain point.
type YuLongCondition struct {
        // Type of the condition.
        Type YuLongConditionType `json:"type"`
        // Status of the condition, one of True, False, Unknown.
        Status corev1.ConditionStatus `json:"status"`
        // The last time this condition was updated.
        // +nullable
        LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
        // Last time the condition transitioned from one status to another.
        // +optional
        // +nullable
        LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
        // The reason for the condition's last transition.
        // +optional
        Reason string `json:"reason,omitempty"`
        // A human readable message indicating details about the transition.
        // +optional
        Message string `json:"message,omitempty"`
}

// YuLongConditionType represents a yu long condition value.
type YuLongConditionType string

const (
        // YuLongReady indicates that the get tidb cluster size.
        YuLongReady YuLongConditionType = "Ready"
)
```


并将该自定义资源注册到schema当中

```golang
// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
        Scheme = scheme
        scheme.AddKnownTypes(SchemeGroupVersion,
                &TidbCluster{},
                &TidbClusterList{},
                &Backup{},
                &BackupList{},
                &BackupSchedule{},
                &BackupScheduleList{},
                &Restore{},
                &RestoreList{},
                &DataResource{},
                &DataResourceList{},
                &TidbInitializer{},
                &TidbInitializerList{},
                &TidbMonitor{},
                &TidbMonitorList{},
                &TidbClusterAutoScaler{},
                &TidbClusterAutoScalerList{},
                &DMCluster{},
                &DMClusterList{},
                &TidbNGMonitoring{},
                &TidbNGMonitoringList{},
                &TidbDashboard{},
                &TidbDashboardList{},
				// 添加我的crd
                &YuLong{},
                &YuLongList{},
        )

        metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
        return nil
}
```

## step 2

开发YuLong的controller，逻辑涉及到的所有文件如下：

```bash
pkg/controller/yulong/yulong_condition_updater.go
pkg/controller/yulong/yulong_control.go
pkg/controller/yulong/yulong_controller.go
pkg/util/yulong/yulong.go
```

## step 3

把YuLong controller添加到tidb-operator的controller-manager当中
cmd/controller-manager/main.go文件修改内容如下：
```golang
diff --git a/cmd/controller-manager/main.go b/cmd/controller-manager/main.go
index 33ba03f11..382a9e9f2 100644
--- a/cmd/controller-manager/main.go
+++ b/cmd/controller-manager/main.go
@@ -173,16 +167,17 @@ func main() {

// Initialize all controllers
controllers := []Controller{
+                       yulong.NewController(deps),
}
```

添加lister到deps当中：
pkg/controller/dependences.go文件修改内容如下：
```shell
diff --git a/pkg/controller/dependences.go b/pkg/controller/dependences.go
index 6c22b8e65..9a5c6b397 100644
--- a/pkg/controller/dependences.go
+++ b/pkg/controller/dependences.go
@@ -239,6 +239,7 @@ type Dependencies struct {
        TiDBMonitorLister           listers.TidbMonitorLister
        TiDBNGMonitoringLister      listers.TidbNGMonitoringLister
        TiDBDashboardLister         listers.TidbDashboardLister
+       YuLongLister                listers.YuLongLister

        // Controls
        Controls
@@ -384,6 +385,7 @@ func newDependencies(
                TiDBMonitorLister:           informerFactory.Pingcap().V1alpha1().TidbMonitors().Lister(),
                TiDBNGMonitoringLister:      informerFactory.Pingcap().V1alpha1().TidbNGMonitorings().Lister(),
                TiDBDashboardLister:         informerFactory.Pingcap().V1alpha1().TidbDashboards().Lister(),
+               YuLongLister:                informerFactory.Pingcap().V1alpha1().YuLongs().Lister(),

                AWSConfig: cfg,
        }, nil
```

## step 4

* 运行编译，make controller-manager
* 打包镜像，make operator-docker
* 发布镜像，make docker-release

