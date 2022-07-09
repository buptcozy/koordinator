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
This proposal describes a structure of the gang with bundles to supports multiple roles under the gang,which is fair for gang scheduling with large workloads.
and also describes a gang scheduling that can be configured in strict mode and non-strict mode,In strict mode, if a pod of the gang fails to pass the Prefilter stage, we ignore the entire bundle task (release occupied resources and wait for the next cycle of scheduling),which is known as All-or-Nothing; In non-strict mode, we try our best to schedule each gang's pod within the specified time.

## Motivation

In AI scenarios, lots of jobs have the need of gang scheduling, that is, after all subtasks are successfully bound, the entire job can run normally,which is the gang scheduling.
We compared with the two gang designs of the community:
### Compared with competitors

#### Coescheduling
(1) Coescheduling schedules a group of gang tasks ,when the cluster resources are not satisfied, it will release all task resources and schedule the gang task in next scheduling cycle.
For example, there is a gang task that requires 10 tasks to be scheduled,when the resources of first 5 tasks are allocated, the 6th task cannot pass Filter for some reason,
Coescheduling will release the first 5 tasks' resource and ignore the remaining 4 tasks in this scheduling cycle.

Our gang scheduling can occupy the resources of first 5 tasks, only ignore the 6th task, then continue scheduling from the 7th task.This is fair for gang with large tasks amount , but it also brings deadlock problems
(Imagine that the resources on a machine are all occupied by two gang tasks, but each gang has not reached the minimum to startup. At this time, both gangs will hold and wait, resulting in a deadlock ),
In the future, we will develop a plugin to solve the gang deadlock.

Therefore, we make this function a configuration item. The user can choose whether to set Strict Mode. In Strict Mode, each bundle's scheduling decisions will be made All Or Nothing , otherwise we will only ignore the pod that failed at Filter point and continue to schedule the remaining pods.

(2) There are often distributed training scenarios like this: Only when a job that wants PS role's resource holding reaches to 50% and the Worker role 's resource holding reaches to %100, the entire job task will be pulled up.
PodGroups cannot be classified according to task roles.

Our gang scheduling will be divided into different bundles according to bundleName,gang can distinguish different roles by bundles, and each bundle can set its own minNum requirements.

(3) When a gang is decided to ignore during this gang scheduling cycle, we should let the remaining pods in the queue to know that they are given up in this gang scheduling cycle.
Coescheduling caches the flag with an expiration time,when the time expires,the flag will be moved from this cache,which means this gang's scheduling cycle is over; we don't think get the scheduling cycle process of a gang based on time is really accurate.
Because it is hard to estimate how long a cycle of scheduling gang is. If the time is set too small, the whole gang scheduling will not be correct; if it is set too large,there will be the waste of time

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

Let's assume a job has two roles: ps and worker, each role has several pods. podA belongs to ps, podB belongs to worker.
if we want bind-condition is ps and worker both reach min-required-number condition, then we can:
```yaml

"metadata": {
  "annotations": {
    "gang.koordinater.io/name" : "job1",
    "gang.koordinater.io/bundleName" : "ps",
    "gang.koordinater.io/waitingTime":"3600s",
    "gang.koordinater.io/minRequiredNumber":"5",
    "gang.koordinater.io/totalChildrenNum":"5",
    "gang.koordinater.io/isStrictMode":"true",
  }
}

"metadata": {
  "annotations": {
    "gang.koordinater.io/name" : "job1",
    "gang.koordinater.io/bundleName" : "worker",
    "gang.koordinater.io/waitingTime":"3600s",
    "gang.koordinater.io/minRequiredNumber":"5",
    "gang.koordinater.io/totalChildrenNum":"5",
    "gang.koordinater.io/isStrictMode":"true",
  }
}

```
the annotations above declare the gang's information:

"job1" gang has two bundles("ps" and "worker"), when each bundle's assumed pods all reach to 5,the whole gang task can start to run;The annotation also declares the gang is in StrictMode and the waitTime is 3600s(the whole gang's schduling valid time).

if we want bind-condition is ps and worker reach min-required-number condition independently, then we can:
```yaml

"metadata": {
  "annotations": {
    "gang.koordinater.io/name" : "job1-ps",
    "gang.koordinater.io/bundleName" : "ps",
    "gang.koordinater.io/waitingTime":"3600s",
    "gang.koordinater.io/minRequiredNumber":"5",
    "gang.koordinater.io/totalChildrenNum":"5",
    "gang.koordinater.io/isStrictMode":"true",
  }
}
"metadata": {
  "annotations": {
    "gang.koordinater.io/name" : "job1-worker",
    "gang.koordinater.io/bundleName" : "ps",
    "gang.koordinater.io/waitingTime":"3600s",
    "gang.koordinater.io/minRequiredNumber":"5",
    "gang.koordinater.io/totalChildrenNum":"5",
    "gang.koordinater.io/isStrictMode":"true",
  }
}
```
this time they belongs to two different gangs:"job1-ps" and "job1-worker",each gang only has one bundle.

### Implementation Details
#### QueueSortPlugin
We design a plugin to implement the QueueSort extension point separately, so that we can integrate the Queue logic of all our plugins and register them in k8s at one time.

This time,we implement the Less function to queue the pods that belong to the same gang together.
The specific queuing rule is:

1.First we compare the priorities of the two pods, the higher priority is at the front of the queue

2.Then we compare the creation time of two pods,if any pod belongs to a gang,then we compare the creation time of the gang, the one created first will be at the front of the queue.

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
  
##### Detail 
###### Data-Structure

1.Gang

We design the gang for record gang status in scheduler memory,it has the "Bundles" field to store the gang's children by bundle name(there is an exampale in API
 section above),and we can also find BundleInfo from "PodToBundleMap" field according to the pod's NamespacedName; We can check the "Satisfied" field to see if the gang is 

2.BundlInfo

We can get the pods from "Children" field,and the "BoundChildren","WaitingForBindChildren" store the pods phase.

We especially explain "ScheduleCycle" and "ChildrenScheduleRoundMap" field.These fields control bundle's scheduling cycle. at the beginning, ScheduleCycle is 1. when each pod comes to pre-filter, we will check if the pod's value in 
ChildrenScheduleRoundMap is euqal to the ScheduleCycle,which means they are scheduled at the same peace. If so, we continue to do the preFilter logic. If they are not euqal, we set the pod's cycle in ChildrenScheduleRoundMap equal with ScheduleCycle.Finally,when the last pod comes to 
make all ChildrenScheduleRoundMap's value equal to ScheduleCycle, ScheduleCycle added by 1, which means a new schedule cycle.And we need to set the ScheduleCycleValid to true.

We continue to explain "ScheduleCycleValid" field,during the scheduling,if it set to "false",means any pod in this bundle shouldn't be scheduled(pods in Permit stage should release the resource ,pods that hasn't come to PreFilter stage will be rejected at PreFilter stage) until it is set to "true". So when a pod failed at Filter, we will set ScheduleCycleValid to false in post-filter, which means 
the remaining pods should be rejected in pre-filter stage. Only When ScheduleCycle added by 1, we will reset the ScheduleCycleValid to true.
```go
type Gang struct {
    Name                 string
    WaitTime             time.Duration 
    Bundles              map[string]*BundleInfo
    PodToBundleMap       map[string]*BundleInfo // podNamespace+"/"+podName
    Satisfied             bool                   // whether the each bundle's assumed pods with the MinRequiredNumber
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
    ScheduleCycleValid           bool
    ChildrenScheduleRoundMap     map[string]int
}
```
3.GangScheduling

this is the framework of the Plugin,we cache the gang info above in the gangCache,and we got the "gangClient" to operate the gang CR.
```go
type GangScheduling struct {
    frameworkHandler            framework.Handle
    gangClient                  gangClient.Interface
    podLister                   listerv1.PodLister
    snapshotSharedLister        framework.SharedLister
    gangCache                   map[string]*Gang
}
```
###### Scheduling Process

during the whole kubernetes shceduling process,we only need to inject our logic into three extention points as below:
```go
var(
     _ framework.PreFilterPlugin = &GangScheduling{}
     _ framework.PostFilterPlugin = &GangScheduling{}
     _ framework.PermitPlugin = &GangScheduling{}
)
type GangScheduling interface{
    ActiveGang(pod *corev1.Pod, state *framework.CycleState)
    PreFilter(context.Context, *corev1.Pod) error
    PostFilter(ctx context.Context, state *CycleState, pod *v1.Pod, filteredNodeStatusMap NodeToStatusMap) (*PostFilterResult, *Status)
    Permit(context.Context, *corev1.Pod) Status
}

```
1.PreFilter

if non-strict-mode, we only do step1 and step2:

(1)Whether the bundle has met the requirements of minNum under each bundle, if not, reject the pod.

(2)Whether the gang has been timeout(check the pod's annotation,introduced at Permit), if yes, reject the pod.

(3)Whether the bundle has met the ScheduleCycleValid check, if not, reject the pod.

(4).Try update ScheduleCycle and ChildrenScheduleRoundMap as mentioned above.


2.PostFilter

At this point means the pod didn't pass the Filter Plugin,we need to decide whether continue scheduling remained children pods of the bundle.

(1)If is strict-mode,we will check the how many pods is in the WaitingForBindChildren map to see if the assumed pods is greater than MinRequiredNumber, if so we will set ScheduleCycleValid to false and release the assumed pods(reject the pods in Permit stage).

(2)If non-strict mode, we will continue to maintain resource occupancy for the assigned pod within the valid time, and continue to schedule subsequent pods to expect that the scheduling conditions will be met.

3.Permit

Any pod passes Filter stage and has already assumed node resource will come to this stage.Scheduler will calculate whether the current number of assumed-pods in each bundle meets the bundle's minNum requirement.

(1)If it is not satisfied, we will give the pod a "Wait" Status with a timeout number(gang's WaitTime),
and later the bind goroutine will keep waiting until the pod is timeout.Then we run the ActiveGang method,it can put all the pods in unscheduablePods or backoffQueue belongs to the gang back to activeQueue after Permit stage,which will make NonStrict Mode more more efficient(no need to wait for the schduler Queue's Loop time)

Let's talk about the pod waiting in the Permit stage,here we consider the gang's scheduling start time is when the first pod comes to the Permit stage.So if the first pod times out later,we will regard the whole gang is time out and 
we will give an annotation like "gang.koordinater.io/timeOut:"true"" to all the pods belong to the gang,then release the resource of all the assumed pods.Before time out,we have chance to wait for the remaining pods to be assumed.

(2)If it is satisfied, we will give each pod in this stage a "Success" status, and we set gang's Satisfied to true, then update gang-status crd, after that any new scheduling event(may be due to reschedule) won't be rejected by gang.


4.Unreserve

We didn't do anything at this stage,since when the pod in Permit stage is timeout or it binds failed will lead the pod to Unreserve stage:

(1)When the pod is timeout,we handle it at Permit stage.


(2)When the pod binds failed, we didn't care about the binding result,because the gang scheduing is responsible for assuming the resource for all pods in each bundle,once they pass the Permit stage means the gang has resource satisfied, and gang's duty is over,
so any pod failed bind,we should only regard the pod as the regular pod next time when it is scheduled,so we want the pod that bind failed and the pod which be preemted handled by the upper level.


5.Init

We will register pod's event handler to watch pod event and update gang and bundle.

#### gang-status crd
gang-status crd is only used for query gang status from outside. the reason is:

1.Easily to query gang-status

2.Record gang bound status in case scheduler failover.Every time when we run the scheduler,we will check the gang crd to rebuild the gang info in the cache.

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
1.User can choose wether gang scheduling in Strict Mode(All-Or-Nothing) or NonStrict Mode,we can compatible in both cases.
## Implementation History
## References