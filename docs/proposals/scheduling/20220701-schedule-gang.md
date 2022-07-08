---
title: gang scheduling
authors:
- "@buptcozy"
  reviewers:
- "@eahydra"
- "@hormes"
- "@allwmh"
- "@honpey"
- "@saintube"
- "@stormgbs"
- "@zwzhang0107"
  creation-date: 2022-07-01
  last-updated: 2022-07-01
  status: provisional

---

# Gang scheduling

## Table of Contents

<!--ts-->

* [Gang scheduling](#Gang-scheduling)
    * [Table of Contents](#table-of-contents)
    * [Summary](#summary)
    * [Motivation](#motivation)
      * [Compared with competitors](#Compared-with-competitors)
        * [Coescheduling](#Coescheduling)
        * [Volcano](#Volcano)
      * [Goals](#goals)
      * [Non Goals and Future Work](#Non-Goals-and-Future-Work)
    * [Proposal](#Proposal)
      * [API](#API)
      * [Implementation Details](#Implementation-Details)
        * [QueueSortPlugin](#QueueSortPlugin)
        * [GangSchedulingPlugin](#GangSchedulingPlugin)
        * [gang crd](#gang-crd)   
    * [Unsolved Problems](#Unsolved-Problems)
    * [Alternatives](#Alternatives)
    * [Implementation History](#Implementation-History)
    * [References](#References)
<!--te-->

## Summary
support more friendly for gang with a large number of tasks

## Motivation

in AI scenario, we may want worker launch at same time.Lots of AI jobs such as tf\pytorch need gang-scheduling

### Compared with competitors

#### Coescheduling
(1) Coescheduling schedules a group of gang tasks ,when the cluster resources are not satisfied, it will release all task resources and schedule the gang task in next scheduling cycle.
For example,  there is a gang task that requires 10 tasks to be scheduled,when the resources of first 5 tasks are allocated, the 6th task cannot pass Filter for some reason,
Coescheduling will release the first 5 tasks' resource and ignore the remaining 4 tasks in this scheduling cycle.

Our gang scheduling can  occupy the resources of first 5 tasks, only ignore the 6th task, then continue scheduling from the 7th task.This is fair for gang  with large tasks amount , but it also brings deadlock problems
(Imagine that the resources on a machine are all occupied by two gang tasks, but each gang has not reached the minimum to startup. At this time, both gangs will hold and wait, resulting in a deadlock ),
In the future, we will develop a plugin to solve the gang deadlock.

Therefore, we make this function a configuration item. The user can choose whether to set Strict Mode. In Strict Mode, scheduling decisions will be made  All Or Nothing , otherwise we will only ignore the pod that failed at Filter point and continue to schedule the remaining pods.

(2) There are often  distributed training scenarios like this: Only when a job that wants PS role's resource holding reaches to 50% and the Worker role 's resource holding reaches to %100, the entire job task will be pulled up.
PodGroups cannot be classified according to task roles.

Our gang scheduling will be divided into different bundles according to bundleName,gang can distinguish different roles by bundles, and each bundle can set its own minNum requirements.

(3) When a gang is decided to ignore during this gang scheduling cycle, we should let the remaining pods in the queue to know that they are given up in this gang scheduling cycle.
Coescheduling caches the flag  with an expiration time,when the time expires,the flag will be moved from this cache,which means this gang's scheduling cycle is over; we don't think  get the scheduling cycle process of a gang based on time  is really accurate.
Because it is hard to estimate how long a cycle of scheduling gang is. If the time is set too small, the whole gang scheduling will not  be correct; if it is set too large,there will be the waste of time

#### Volcano
volcano uses JobInfo as the basic unit for scheduling, in which the scheduler is very different from the community from the data structure to the process, and every time the function is expanded, data conversion is performed, and the connection with the community is not good.

### Goals
1.propose some pod's annotations to announce gang properties

2.design a gang-crd to display gang-status

3.scheduler should provide gang data-structure and schedule-plugin

### Non Goals and Future Work
1.Provide deadlock plugins to solve gang-resource-deadlock problems

## Proposal
### API
we choose to create gang-related by pod's annotations instead of gang crd. This is because we don't want high level
operator to maintain gang-crd's life circle. meanwhile, it's inconvenient to maintain receive-order-issue's between
gang-crd and pod in scheduler side.That is to say:

1.From a Scheduler perspective, when pods are created before gang, we can't find any other information about this gang at this time.

2.From an Operator perspective, we need to do additional work to update/create/delete the gang

```go
labels:
    gang.koordinater.io/name: job1
    gang.koordinater.io/bundleName:ps
    gang.koordinater.io/waitingTime:3600
    gang.koordinater.io/minRequiredNumber:5
```

let's assume a job has two roles: ps and worker, each role has several pods. podA belongs to ps, podB belongs to worker.
if we want bind-condition is ps and worker both reach min-required-number condition, then we can:
```go
podA.Annotation[GangName] = "job1"
podA.Annotation[GangBundleName] = "ps"
podA.Annotation[GangWaitTime] = "3600s"
podA.Annotation[GangBundleMinRequiredNumber] = 5
podA.Annotation[TotalChildrenNum] = 5

podB.Annotation[GangName] = "job1"
podB.Annotation[GangBundleName] = "worker"
podB.Annotation[GangWaitTime] = "3600s"
podB.Annotation[GangBundleMinRequiredNumber] = 5
podB.Annotation[TotalChildrenNum] = 5
```

if we want bind-condition is ps and worker reach min-required-number condition independently, then we can:
```go
podA.Annotation[GangName] = "job1-ps"
podA.Annotation[GangBundleName] = "ps"
podA.Annotation[GangWaitTime] = "3600s"
podA.Annotation[GangBundleMinRequiredNumber] = 5
podA.Annotation[TotalChildrenNum] = 5

podB.Annotation[GangName] = "job1-worker"
podB.Annotation[GangBundleName] = "worker"
podB.Annotation[GangWaitTime] = "3600s"
podB.Annotation[GangBundleMinRequiredNumber] = 5
podB.Annotation[TotalChildrenNum] = 5
```
### Implementation Details
#### QueueSortPlugin
We design a plugin to implement the QueueSort extension point separately, so that we can integrate the Queue logic of all our plugins and register them in k8s at one time.

This time,we implement the Less function to arrange pods that belong to the same gang together.
Specifically, on the basis of priority queuing，we set the following 3 rules:

1.If both Pods are regularPods (ordinary Pods), the one  created firstly will be ranked first in the queue

2.If one of the two Pods is a regularPod and the other is a gangPod (a Pod belonging to a Gang), we compare the creation time of the regularPod and the creation time of the Gang  which the gangPod belongs, the one created firstly will be ranked first in the queue.

3.If both Pods are gangPods, we compare the creation times of the two gangs, the one created firstly will be ranked first in the queue. It is possible that the creation time of two Gangs is the same, and we compare their NamespacedName (the purpose here is to distinguish different Gangs).
```go
type QueueSortPlugin interface{
    QueueSort(*QueuedPodInfo, *QueuedPodInfo) bool
}
```

#### GangSchedulingPlugin
##### target

1.We should create a new gang-plugin to implement PreFilter\PostFilter\Permit\UnReserve stage. in this plugin, we should
   provide gang-cache to store data and gang-controller to monitor gang status.

2.We should implement gang-crd update\recover logic in gang-plugin
  
##### detail
We introduce Gang and BundleInfo for record gang status in scheduler memory. We especially explain ScheduleCycle and ChildrenScheduleRoundMap.
These variables control bundle's scheduling cycle. at the beginning, ScheduleCycle is 1. when each pod comes to pre-filter, we will check pod's value in 
ChildrenScheduleRoundMap is euqal to the  ScheduleCycle which means they are scheduled  at the same peace. If so, we continue to do the preFilter logic. If they are  not euqal, we set the pod's cycle in ChildrenScheduleRoundMap equal with ScheduleCycle. Finally,  when the last pod comes to 
make all ChildrenScheduleRoundMap's value equal to ScheduleCycle, ScheduleCycle added by 1, which means a new schedule cycle.

We continue to explain ScheduleCycleValid. in strict-mode, when a pod failed for scheduling, we will set ScheduleCycleValid to false in post-filter, which means in 
this schedule cycle, the remaining pods should be rejected in pre-filter stage. when ScheduleCycle added by 1, we will reset the ScheduleCycleValid to true.
```go
type Gang struct {
    Name                 string
    WaitTime             time.Duration 
    Timer                *util.CancellableTimer // timer to rollback allocs of gang
    Bundles              map[string]*BundleInfo
    PodToBundleMap       map[string]*BundleInfo // podNamespace+"/"+podName
    HasBound             bool                   // whether is bound
    IsStrictMode         bool                   //whether is in strictMode
}

type BundleInfo struct {
    Name                        string
    Children                    map[string]*PodInfo  
    BoundChildren               map[string]*PodInfo
    WaitingForBindChildren      map[string]*PodInfo
    MinRequiredNumber           int
    TotalChildrenNum            int

    ScheduleCycle                int
    ScheduleCycleInValid           bool
    ChildrenScheduleRoundMap     map[string]int
}

type GangScheduling struct {
    frameworkHandler            framework.Handle
    gangClient                  gangClient.Interface
    podLister                   listerv1.PodLister
    snapshotSharedLister        framework.SharedLister
    podSchedulingCycleCache     map[string]int
    gangCache                   map[string]*Gang
}

type GangScheduling interface{
    MonitorLoop()
    PreFilter(context.Context, *corev1.Pod) error
    PostFilter(ctx context.Context, state *CycleState, pod *v1.Pod, filteredNodeStatusMap NodeToStatusMap) (*PostFilterResult, *Status)
    Permit(context.Context, *corev1.Pod) Status
}
```

1.PreFilter

if non-strict-mode, we only do step1 and step2:

i.Whether the bundle has met the requirements of minNum under each bundle, if not, reject the pod.

ii.Whether the gang has been timeout, if yes, reject the pod.
      
iii.Whether the bundle has met the ScheduleCycleValid check, if not, reject the pod.
    
iv.Try update ScheduleCycle and ChildrenScheduleRoundMap as mentioned above.


2.PostFilter

At this point means the pod didn't pass the Filter Plugin,we need to decide whether continue scheduling remained children pods of gang.

i.If is strict-mode, when the number of assigned pods in each bundle cannot meet the minNum requirement, we will set ScheduleCycleValid false and release the assumed pods.

ii.If non-strict mode, we will continue to maintain resource occupancy for the assigned pod within the valid time, and continue to schedule subsequent pods to expect that the scheduling conditions will be met.

3.Permit

i.Scheduler will calculate whether the current number of assigned-pods in each bundle meets the bundle minNum requirement. If it is satisfied, it will be passed to bind,
and we set gang bound to true, then update gang-status crd, after that any new scheduling event(may be due to reschedule) won't be rejected by gang.

ii.If it didn't meet the expected requirement, in non-strict-mode we will keep the pods continuous waiting until the gang is time out. 
In strict-mode we will release the assumed pods.

4.Unreserve

Both timeout and bind failure will lead the pod to Unreserve,we need to handle it  separately:

(1)When the pod is timeout,we need to release the resource of the assined pods and set the ScheduleCycleInValid of the gang to true ,which make the remaining pods not pass the PreFilter stage,then we will patch a timeout annotation to each gang's pod, and these pod won't be scheduled again.


(2)When the pod binds failed, In Strict Mode, we will release the resource of the assined pods and set the ScheduleCycleInValid of the gang to true;

In NonStrict Mode,we do nothing but continue scheduling the remaining pods.


5.Init

We will register pod's event handler to watch pod event and update gang and bundle.

#### gang-status crd
gang-status crd is only used for query gang status from outside. the reason is:

1.Easily to query gang-status

2.Record gang bound status in case scheduler failover.

3.When all pod deleted, we will delete gang-crd automatically.
   
```go
type Gang struct {
    metav1.TypeMeta
    metav1.ObjectMeta
    Spec   GangSpec
    Status GangStatus
}
type GangSpec struct {
    Bundles           []BundleSpec         `json:"bundles,omitempty"`
    WaitTime          *metav1.Duration `json:"waitTime,omitempty"`
    IsStrictMode      bool
}
type GangStatus struct {
    ResourceSatisfied bool
    schedulingCycle   int64
}
type BundleSpec struct {
    Name string 
    MinRequiredNumber int32
}
```

## Unsolved Problems
## Alternatives
## Implementation History
## References