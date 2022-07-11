---
title: gang scheduling
authors:
- "@buptcozy"
  reviewers:
- "@eahydra"
- "@hormes"
- "@yihuifeng"
- "@honpey"
- "@zwzhang0107"
- "@jasonliu747"
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
        * [Data-Structure](#data-structure)
        * [GangPlugin](#gang-plugin)
    * [Unsolved Problems](#Unsolved-Problems)
    * [Alternatives](#Alternatives)
    * [Implementation History](#Implementation-History)
    * [References](#References)
<!--te-->

## Summary
This proposal provides gang mechanism for the scheduler to control pods binding opportunity. User can declare a resource-collection-minimum number, 
only when assigned-resources reach the given limitation can trigger the binding. We provide `strict-mode` and `non-strict-mode` to 
control the resource-accumulation-process by a configuration. We also provide a two-level gang description for better matching 
the real scenario, which is different from community.

## Motivation
In AI scenarios, lots of jobs need gang scheduling. The community have lots of related implements such `coescheduling` or `vocalno`.
However, in practice we find that the above solutions do not meet the needs of the business in some places.

### Compared with competitors

#### Coescheduling
1. `coescheduling` implement a new queue-sort interface and other methods to let one gang's pods get out of the queue in order as much as possible.
If a pod failed to be scheduled, the requests that have been successfully scheduled in this round of gang scheduling cycle will be rolled back,
and the remaining pods waiting for scheduling will be rejected in pre-filter check util this scheduling cycle passed. 
For example, there a gang requires 10 tasks to be scheduled, if first 5 tasks allocated, the 6th task failed to be scheduled,
`coescheduling` will roll-back first 5 tasks and ignore the remaining 4 tasks in this gang scheduling cycle. `coescheduling` simply use a 
global time interval to control the gang scheduling cycle. The first defect is that the uniform time interval will cause 
some problems. If the time configuration is too long, it will lead to useless waiting; If the time configuration is too short, 
it will lead to useless scheduling. Secondly, it is very difficult for a large job to meet all resource requests at one time. 
This mechanism will lead to a very low probability of full resources, and eventually make the job starve to death. We call this process as `strict-mode`.

2. Some jobs have complex gang requirements. For example, a job has several roles. Each role will have several pods 
and its own gang conditions. Jobs also need different roles to form different gang groups. All pods in a gang group can 
trigger the bind process only after all roles in a gang group meet their gang conditions. The `coescheduling` can't meet
this requirement.

#### Volcano
volcano uses JobInfo as the basic unit for scheduling and is naturally friendly to gang. However, the scheduler is very 
different from the community from the data structure to the process, which lead negative compatibility with the native k8s community.

### Goals
1. Definition API to announce gang-scheduling-configuration.

2. Provides a scheduler plugin to achieve gang-scheduling ability.

### Non Goals and Future Work
1. Provide ability to solve gang-resource-deadlock problems with `non-strict-mode`.

## Proposal
### API
We advise users declaring gang-scheduling-configuration by pod's annotation. First Reason, high level operator has no need 
to maintain gang-crd's life circle, for example handle `update/create/delete` events. Second Reason, from a Scheduler perspective, 
it's inconvenient to maintain receive-order-issue's between gang-crd and pod. According to practical experience, pod's annotation 
is enough for declaring gang-scheduling-configuration,so we build the gang's information according to the first pod which declared the gang.

- `koordinater.io.gang/name`                gang name.
- `koordinater.io.gang/bundleName`          equals to roleName.
- `koordinater.io.gang/waitingTime`         max wait time since first pod comes to permit stage.
- `koordinater.io.gang/minRequiredNumber`   the minimum pod's number satisfying bundle condition.
- `koordinater.io.gang/totalChildrenNumber` the total pod's number in a bundle.
- `koordinater.io.gang/isStrictMode`        work in `strict-mode` or `non-strict-mode`.
- `koordinater.io.gang/ganggroups`          describe gang groups.

Let's assume a job has two roles: A and B, each role has several pods. podA belongs to roleA, podB belongs to roleB.
roleA and roleB belongs to one gang group, the example as follows:
```yaml
"metadata": {
  "annotations": {
    "koordinater.io.gang/name" : "jobA",
    "koordinater.io.gang/bundleName" : "roleA",
    "koordinater.io.gang/waitingTime":"3600s",
    "koordinater.io.gang/minRequiredNumber":"5",
    "koordinater.io.gang/totalChildrenNumber":"5",
    "koordinater.io.gang/isStrictMode":"true",
    "koordinater.io.gang/ganggroups":"[[roleA, roleB]]",
  }
}

"metadata": {
  "annotations": {
    "koordinater.io.gang/name" : "jobB",
    "koordinater.io.gang/bundleName" : "roleB",
    "koordinater.io.gang/waitingTime":"3600s",
    "koordinater.io.gang/minRequiredNumber":"5",
    "koordinater.io.gang/totalChildrenNum":"5",
    "koordinater.io.gang/isStrictMode":"true",
    "koordinater.io.gang/ganggroups":"[[roleA, roleB]]",
  }
}
```

If assume a job has two roles: A and B, each role has several pods. podA belongs to roleA, podB belongs to roleB.
roleA and roleB belongs to different gang group, the example as follows:
```yaml
"metadata": {
  "annotations": {
    "koordinater.io.gang/name" : "jobA",
    "koordinater.io.gang/bundleName" : "roleA",
    "koordinater.io.gang/waitingTime":"3600s",
    "koordinater.io.gang/minRequiredNumber":"5",
    "koordinater.io.gang/totalChildrenNumber":"5",
    "koordinater.io.gang/isStrictMode":"true",
    "koordinater.io.gang/ganggroups":"[[roleA], [roleB]]",
  }
}

"metadata": {
  "annotations": {
    "koordinater.io.gang/name" : "jobB",
    "koordinater.io.gang/bundleName" : "roleB",
    "koordinater.io.gang/waitingTime":"3600s",
    "koordinater.io.gang/minRequiredNumber":"5",
    "koordinater.io.gang/totalChildrenNum":"5",
    "koordinater.io.gang/isStrictMode":"true",
    "koordinater.io.gang/ganggroups":"[[roleA], [roleB]]",
  }
}
```

### Implementation Details
#### QueueSortPlugin

We design a independent plugin to implement the `QueueSort` extension point separately, so that we can integrate 
queue sort logic of all plugins, and register them at one time.

In this proposal, we implement the Less function to gather pods belongs to same gang. The specific queuing rule is:

1. Firstly, compare the priorities of the two pods, the higher priority is at the front of the queue.

2. Secondly, compare create-time-stamp of two pods, if pod belongs to a gang, then we compare create-time-stamp of the gang, 
the one created first will be at the front of the queue.

3. Finally, compare pod's namespace, if pod belongs to a gang, then we compare gang name. 

```go
type QueueSortPlugin interface{
    QueueSort(*QueuedPodInfo, *QueuedPodInfo) bool
}
```

#### GangSchedulingPlugin
##### Data-Structure

###### strict-mode and non-strict-mode
As mentioned above, in `strict-mode`, if a pod failed to be scheduled, the requests that have been successfully scheduled in 
this scheduling cycle will be rolled back, and the remaining pods waiting for scheduling will be rejected in 
pre-filter check util this scheduling cycle passed. We call this mode is `strict-mode`.

In `non-strict-mode`, if a pod failed to be scheduled, it has no impact on any other pod. We will continue to accumulate 
the allocated pod until the condition of gang is met. This process is friendly to gangs with large number of pods, but it 
will increase the risk of resource deadlock between gangs. For example, the quota of the quota group is 10(quota will be proposed later), 
and the user submits three gangs with 5 pods. Due to various plugin constraints, gang1\2\3 may allocate resources of 3\3\4 respectively. 
Since the quota group's quota is full, there will be no new resource scheduling. We call this is resource deadlock of resource gang.
In future proposal, we will try to fix this problem.

###### Gang

We design the gang to record gang status in scheduler memory, it has the `Bundles` field to store the gang's children by bundle name.
We can also find BundleInfo from `PodToBundleMap` field according to the pod's NamespacedName. We can check the `ResourceSatisfied`
field to see if the gang is already has the minimum number assumed pods in each bundle. 

It should be noted that, if the resource accumulation conditions of gang are met, then some pods failed in the process of binding;
Or some bound pods are preempted\rescheduled, should the constraints of gang still be effective in the process of resource reallocation? 
Because the initial purpose of gang is to require pods to be pulled up at the same time, if some pods have been pulled up, 
then the subsequent gang behavior is meaningless. Therefore, when `resourcequalified=true`, all subsequent resource allocations 
are no longer constrained by gang rules, and their performance is similar to ordinary pod.

As mentioned above, `WaitTime` is the max wait time since first pod comes to permit stage. if `WaitTime` is timeout, 
scheduler will roll back all assumed pods, update each pod's annotation with `koordinater.io.gang/timeout=true`, and
won't schedule these pods anymore. User should pay attention to this status and delete pods timely.

```go
type Gang struct {
    Name                 string                 //gang name
    WaitTime             time.Duration          //max wait time since first pod comes to permit stage.
    Bundles              map[string]*BundleInfo 
    PodToBundleMap       map[string]*BundleInfo //podNamespace+"/"+podName
    ResourceSatisfied    bool                   //whether the each bundle's assumed pods with the MinRequiredNumber
    IsStrictMode         bool                   //whether is in strict-mode
    GangGroup            map[string][]string    //gang group
}
```

###### BundleInfo

We can get the children pods from "Children" field, and the `BoundChildren, WaitingForBindChildren` store the pods binding status,
which is used to check if the pods can pass permit stage.

We especially explain `scheduleCycle` and `childrenScheduleRoundMap` field. These fields control bundle's scheduling cycle. For example,
at the beginning, `scheduleCycle` is 1, and each pod's cycle in `childrenScheduleRoundMap` is 0. When each pod comes to pre-filter, 
we will check if the pod's value in `childrenScheduleRoundMap` is smaller than bundle's `scheduleCycle`, If result is positive, 
we set the pod's cycle in `childrenScheduleRoundMap` equal with `scheduleCycle` and pass the check. If result is negative, means
the pod has been scheduled in this cycle, so we should reject it. When the last pod comes to make all `childrenScheduleRoundMap`'s values
equal to `scheduleCycle`, bundle's `scheduleCycle` will be added by 1, which means a new schedule cycle.

We continue to explain `scheduleCycleValid` field, during the scheduling,  When a pod failed at Filter stage, we will set ScheduleCycleValid to 
false in Post-Filter stage, which means any pod in this bundle shouldn't be scheduled until it is set to "true".
the remaining pods should be rejected in pre-filter stage. Only When `scheduleCycle` added by 1, we will reset the `scheduleCycleValid` to true.

It should be emphasized that `scheduleCycle\scheduleCycleValid\childrenScheduleRoundMap` only work in `strict-mode`. 

```go
type BundleInfo struct {
    Name                         string
    Children                     map[string]*PodInfo  
    BoundChildren                map[string]*PodInfo
    WaitingForBindChildren       map[string]*PodInfo
    MinRequiredNumber            int
    TotalChildrenNum             int

    ScheduleCycle                int
    ScheduleCycleValid           bool
    ChildrenScheduleRoundMap     map[string]int
}
```

##### GangPlugin

this is the framework of the Plugin,we cache the gang info above in the gangCache,and we got the "gangClient" to operate the gang CR.
```go
type GangPlugin struct {
    frameworkHandler            framework.Handle
    gangClient                  gangClient.Interface
    podLister                   listerv1.PodLister
    snapshotSharedLister        framework.SharedLister
    gangCache                   map[string]*Gang
}
```
during the whole kubernetes shceduling process,we only need to realize our logic into four extention points as below:
```go
var(
     _ framework.PreFilterPlugin = &GangScheduling{}
     _ framework.PostFilterPlugin = &GangScheduling{}
     _ framework.PermitPlugin = &GangScheduling{}
     _ framework.ReservePlugin = &Coscheduling{}
)

type GangScheduling interface{
    ActiveGang(pod *corev1.Pod, state *framework.CycleState)
    PreFilter(context.Context, *corev1.Pod) error
    PostFilter(ctx context.Context, state *CycleState, pod *v1.Pod, filteredNodeStatusMap NodeToStatusMap) (*PostFilterResult, *Status)
    Permit(context.Context, *corev1.Pod) Status
    Unreserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string)
}

```
###### **PreFilter**

if `non-strict-mode`, we only do step1 and step2:

- Check whether childrens in bundle has met the requirements of minimum number under each bundle, and reject the pod if negative.

- Check whether the gang has been timeout(check the pod's annotation,later introduced at Permit section), and reject the pod if positive.

- Check whether the bundle has met the `scheduleCycleValid` check, and reject the pod if negative.

- Try update `scheduleCycle`, `scheduleCycleValid`, `childrenScheduleRoundMap` as mentioned above.


###### **PostFilter**

At this point means the pod didn't pass the Filter Plugin, we should:

- If `strict-mode`, we will set `scheduleCycleValid` to false and release all assumed pods.

- If `non-strict mode`, we will do nothing.

###### **Permit**

Any pod passes Filter stage will come to this stage. Scheduler will calculate whether the current number of assumed-pods 
in each bundle meets the bundle's minimum requirement.

- If gang don't meet the bind-condition, we will give the pod a "Wait" Status with a timeout duration, and the bind 
goroutine will keep waiting until the wait is timeout or passed. Then we will run the `ActiveGang` method, it can put all 
the pods belong to the gang which in `schedulableQueue` or `backoffQueue` back to `activeQueue`, so that the pod of gang 
can be continuously scheduled as much as possible.

- If gang meet the bind-condition, we will give every waiting pod a "Success" status, which will let the bind goroutine of
each pod leave the waiting status and continue to run. Also, as mentioned above, we will set gang's `ResourceSatisfied` to true.

###### **Un-reserve**

Both permit stage is timeout and binding failed will lead the pod to un-reserve stage, we can distinguish from gang's "ResourceSatisfied" field,
if the field is true means binding failed, else means the gang is timeout.

- When permit stage is timeout, we will give an annotation like `koordinater.io.gang/timeout=true` to all the pods 
belong to the gang and will release the resource of all the assumed pods. The gang will not be scheduled anymore, 
user should manually handle the timeout event.

- When binding failed, as mentioned above, the collection of gang's resource is over, we will do nothing except roll back
the failed pod resource.

###### **Init**

We will register pod's event handler to watch pod event for updating gang and bundle.


## Unsolved Problems

## Alternatives
User can choose use gang by `strict-mode` and `non-strict-mode` case by case.

## Implementation History

## References