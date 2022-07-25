package gang

import (
	"context"
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"time"
)

type Status string

const (
	Name = "Gang"
)

const (
	GangWaitTime = 2 * time.Minute

	StrictMode    = "StrictMode"
	NonStrictMode = "NonStrictMode"

	//  Annouce Gang annotaion
	GangNameAnnotation     = "gang.scheduling.koordinator.sh/name"
	GangMinNumAnnotaion    = "gang.scheduling.koordinator.sh/min-available"
	GangWaitTimeAnnotaion  = "gang.scheduling.koordinator.sh/waiting-time"
	GangTotalNumAnnotation = "gang.scheduling.koordinator.sh/total-number"
	GangModeAnnotation     = "gang.scheduling.koordinator.sh/gang-mode"
	GangGourpsAnnotation   = "gang.scheduling.koordinator.sh/groups"
	GangTimeOutAnnotation  = "gang.scheduling.koordinator.sh/timeout"

	//Permit internal status
	GangNotFoundInCache Status = "Gang not found in cache"
	Success             Status = "Success"
	Wait                Status = "Wait"
)

var (
	_ framework.PreFilterPlugin  = &GangPlugin{}
	_ framework.PostFilterPlugin = &GangPlugin{}
	_ framework.ReservePlugin    = &GangPlugin{}
	_ framework.PostBindPlugin   = &GangPlugin{}
	_ framework.PostBindPlugin   = &GangPlugin{}
)

type GangPlugin struct {
	frameworkHandler framework.Handle
	podLister        v1.PodLister
	gangCache        *gangCache
}

func New(args runtime.Object, handle framework.Handle) (framework.Plugin, error) {
	gangCache := NewGangCache()
	//recover the gangCache
	if err := RecoverGangCache(handle, gangCache); err != nil {
		return nil, err
	}
	podInformer := handle.SharedInformerFactory().Core().V1().Pods().Informer()
	podLister := handle.SharedInformerFactory().Core().V1().Pods().Lister()
	podInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		//FilterFunc: func(obj interface{}) bool {
		//	switch t := obj.(type) {
		//	case *corev1.Pod:
		//		return CheckPodGangInfo(t)
		//	default:
		//		utilruntime.HandleError(fmt.Errorf("unable to handle object %T", obj))
		//		return false
		//	}
		//},
		Handler: cache.ResourceEventHandlerFuncs{

			AddFunc:    gangCache.onPodAdd,
			DeleteFunc: gangCache.onPodDelete,
		},
	})
	return &GangPlugin{
		frameworkHandler: handle,
		podLister:        podLister,
		gangCache:        gangCache,
	}, nil
}

func (p *GangPlugin) Name() string { return Name }

// PreFilter
//if non-strict-mode, we only do step1 and step2:
// i.Check whether childes in Gang has met the requirements of minimum number under each Gang, and reject the pod if negative.
// ii.Check whether the Gang has been timeout(check the pod's annotation,later introduced at Permit section), and reject the pod if positive.
// iii.Check whether the Gang has met the scheduleCycleValid check, and reject the pod if negative.
// iv.Try update scheduleCycle, scheduleCycleValid, childrenScheduleRoundMap as mentioned above.
func (p *GangPlugin) PreFilter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod) *framework.Status {
	gangCache := p.gangCache
	gangName := pod.Annotations[GangNameAnnotation]
	mode, found := gangCache.GetGangMode(gangName)
	if !found {
		klog.Infof("pre-filter pod %v  from Gang %v rejected,didn't find the Gang in the cache ", pod.Name, gangName)
		return framework.NewStatus(framework.Unschedulable, "can not find gang in the gang cache")
	}
	if err := p.PreFilterCheck(pod, gangName, mode); err != nil {
		klog.Errorf("PreFilter failed err:%s", err.Error)
		return framework.NewStatus(framework.UnschedulableAndUnresolvable, err.Error())
	}
	return framework.NewStatus(framework.Success, "")
}

// PreFilterExtensions returns a PreFilterExtensions interface if the plugin implements one.
func (p *GangPlugin) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// PostFilter
//i. If strict-mode, we will set scheduleCycleValid to false and release all assumed pods.
//ii. If non-strict mode, we will do nothing.
func (p *GangPlugin) PostFilter(ctx context.Context, state *framework.CycleState, pod *corev1.Pod,
	filteredNodeStatusMap framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	gangCache := p.gangCache
	gangName := pod.Annotations[GangNameAnnotation]
	mode, found := gangCache.GetGangMode(gangName)
	if !found {
		klog.Infof("pre-filter pod %v  from Gang %v rejected,didn't find the Gang in the cache ", pod.Name, gangName)
		return &framework.PostFilterResult{}, framework.NewStatus(framework.Unschedulable, "can not find gang in the gang cache")
	}
	if mode == StrictMode {
		p.frameworkHandler.IterateOverWaitingPods(func(waitingPod framework.WaitingPod) {
			if waitingPod.GetPod().Annotations[GangNameAnnotation] == gangName {
				klog.Errorf("postFilter rejects the pod name:%v from Gang %s", pod.Name, gangName)
				waitingPod.Reject(p.Name(), "optimistic rejection in PostFilter")
			}
		})
		gangCache.SetScheduleCycleValid(gangName, false)
		return &framework.PostFilterResult{}, framework.NewStatus(framework.Unschedulable,
			fmt.Sprintf("Gang %v gets rejected this cycle due to Pod %v is unschedulable even after PostFilter in StrictMode", gangName, pod.Name))
	}
	return &framework.PostFilterResult{}, framework.NewStatus(framework.Unschedulable,
		fmt.Sprintf("Pod %v from Gang %v is unschedulable in NonStrictMode", gangName, pod.Name))
}

// Permit
//we will calculate all Gangs in GangGroup whether the current number of assumed-pods in each Gang meets the Gang's minimum requirement.
//and decide whether we should let the pod wait in Permit stage or let the whole gangGroup go binding
func (p *GangPlugin) Permit(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) (*framework.Status, time.Duration) {
	waitTime, s := p.PermitCheck(pod)
	var retStatus *framework.Status
	switch s {
	case GangNotFoundInCache:
		return framework.NewStatus(framework.Unschedulable, "Gang not found in gangCache"), 0
	case Wait:
		klog.Infof("Pod %v from gang %v is waiting to be scheduled at Permit stage", pod.Name, pod.Annotations[GangNameAnnotation])
		retStatus = framework.NewStatus(framework.Wait)
		p.ActivateGang(pod, state)
	case Success:
		p.AllowGangGroup(pod)
		retStatus = framework.NewStatus(framework.Success)
		waitTime = 0
	}
	return retStatus, waitTime
}

// Reserve is the functions invoked by the framework at "reserve" extension point.
func (p *GangPlugin) Reserve(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) *framework.Status {
	return nil
}

// Unreserve
//(1)handle the timeout gang
//(2)do nothing when bound failed
func (p *GangPlugin) Unreserve(ctx context.Context, state *framework.CycleState, pod *corev1.Pod, nodeName string) {
	gangName := pod.Annotations[GangNameAnnotation]
	gangCache := p.gangCache
	resourceSatisfied, _ := gangCache.IsGangResourceSatisfied(gangName)

	//gang time out
	if !resourceSatisfied {
		klog.Infof("gang %v is time out,start to release the assumed resource and add annotations to the gang's children")
		timeoutAnnotations := map[string]interface{}{
			"metadata": map[string]map[string]string{
				"Annotations": {
					GangTimeOutAnnotation: "true",
				}},
		}
		pods, err := p.podLister.List(nil)
		if err != nil {
			klog.Errorf("unReserve list pod err : %v", err.Error())
			return
		}
		//add timeout annotation to all the children of the gang
		for _, pod := range pods {
			if pod.Annotations[GangNameAnnotation] == gangName {
				ns := pod.Namespace
				podName := pod.Name
				updateAnnotation, _ := json.Marshal(timeoutAnnotations)
				_, err := p.frameworkHandler.ClientSet().CoreV1().Pods(ns).Patch(ctx, podName, types.StrategicMergePatchType, updateAnnotation, metav1.PatchOptions{})
				if err != nil {
					klog.Errorf("unReserve when patch annotation to pod err : %v", err.Error())
				}
			}
		}
		//release resource of all assumed children of the gang
		p.frameworkHandler.IterateOverWaitingPods(func(waitingPod framework.WaitingPod) {
			if waitingPod.GetPod().Annotations[GangNameAnnotation] == gangName {
				klog.Errorf("unReserve rejects the pod name:%v from Gang %s due to timeout", pod.Name, gangName)
				waitingPod.Reject(p.Name(), "optimistic rejection in unReserve due to timeout")
			}
		})
	}
	return
}

// PostBind just update the gang's BoundChildren
func (p *GangPlugin) PostBind(ctx context.Context, _ *framework.CycleState, pod *corev1.Pod, nodeName string) {
	gangCache := p.gangCache
	gangCache.AddBoundPod(pod)
	return
}

func RecoverGangCache(handle framework.Handle, gangCache *gangCache) error {
	podLister := handle.SharedInformerFactory().Core().V1().Pods().Lister()
	podsList, err := podLister.List(nil)
	if err != nil {
		klog.Errorf("RecoverGangCache podsList List error %+v", err)
		return err
	}
	for _, pod := range podsList {
		if pod.Annotations[GangNameAnnotation] != "" {
			gangCache.onPodAdd(pod)
			if pod.Spec.NodeName != "" {
				//todo:没想好如何区分assumedpod 和 boundpod，暂时先按assumed处理，不影响permit计数
				gangCache.AddAssumedPod(pod)
			}
		}
	}
	//todo:严格模式下 schedulingCycle 如何recover呢？
	return nil
}

func (p *GangPlugin) PreFilterCheck(pod *corev1.Pod, gangName string, mode string) error {
	gangCache := p.gangCache
	var currentChildrenNum int
	var minRequireChildrenNum int
	var gangScheduleCycle int
	var podScheduleCycle int
	var found bool
	//check if reach MinNumber
	if currentChildrenNum, found = gangCache.GetChildrenNum(gangName); !found {
		return fmt.Errorf("pre-filter pod %v  from Gang %v rejected,didn't find the Gang in the cache ", pod.Name, gangName)
	}
	if minRequireChildrenNum, found = gangCache.GetGangMinNum(gangName); !found {
		return fmt.Errorf("pre-filter pod %v  from Gang %v rejected,didn't find the Gang in the cache ", pod.Name, gangName)
	}

	if currentChildrenNum < minRequireChildrenNum {
		return fmt.Errorf("pre-filter pod %v cannot find enough children pods from Gang %v, "+
			"current children number: %v, minRequiredNumber of Gang is %v", pod.Name, gangName, currentChildrenNum, minRequireChildrenNum)
	}
	//check if Gang is timeout
	if pod.Annotations[GangTimeOutAnnotation] == "true" {
		return fmt.Errorf("pre-filter pod %v from Gang %v rejected,Gang is timeout", pod.Name, gangName)
	}

	if mode == StrictMode {
		if gangScheduleCycle, found = gangCache.GetGangScheduleCycle(gangName); !found {
			return fmt.Errorf("pre-filter pod %v  from Gang %v rejected,didn't find the Gang in the cache ", pod.Name, gangName)
		}
		if podScheduleCycle, found = gangCache.GetChildScheduleCycle(gangName, pod.Name); !found {
			return fmt.Errorf("pre-filter pod %v  from Gang %v rejected,didn't find the Gang in the cache ", pod.Name, gangName)
		}
		//firstly, filter the pods whose cycle is greater than GangScheduleCycle,
		//Actually,there shouldn't be the greater condition,at most a pod is scheduled twice in this gangScheduleCycle
		//So we don't add it's podCycle,remaining equal with gangScheduleCycle
		if podScheduleCycle >= gangScheduleCycle {
			klog.Errorf("pre-filter pod's cycle is greater than GangScheduleCycle", pod.Name, gangName)
		}
		//secondly, set the pod's cycle equal with gangScheduleCycle
		gangCache.SetChildCycle(gangName, pod.Name, gangScheduleCycle)
		//check the if gang's cycle valid
		if valid, found := gangCache.IsGangScheduleCycleValid(gangName); !found {
			return fmt.Errorf("pre-filter pod %v  from Gang %v rejected,didn't find the Gang in the cache ", pod.Name, gangName)
		} else {
			if !valid {
				return fmt.Errorf("pre-filter pod %v from Gang %v rejected,Gang's ScheduleCycle is not valid", pod.Name, gangName)
			}
		}
		//finally, check if all the pods in this gangScheduleCycle has been handled
		if gangTotalNum, found := gangCache.GetGangTotalNum(gangName); !found {
			return fmt.Errorf("pre-filter pod %v  from Gang %v rejected,didn't find the Gang in the cache ", pod.Name, gangName)
		} else {
			if gangCache.CountChildNumWithCycle(gangName, gangScheduleCycle) == gangTotalNum {
				gangCache.SetScheduleCycleValid(gangName, true)
				gangCache.SetScheduleCycle(gangName, gangScheduleCycle+1)
			}
		}
	}
	return nil
}

func (p *GangPlugin) PermitCheck(pod *corev1.Pod) (time.Duration, Status) {
	gangName := pod.Annotations[GangNameAnnotation]
	gangCache := p.gangCache
	waitTime, found := gangCache.GetGangWaitTime(gangName)
	if !found {
		return 0, GangNotFoundInCache
	}
	//first we need to add the pod to assumedMap of gang
	gangCache.AddAssumedPod(pod)
	gangGroup, _ := gangCache.GetGangGroup(gangName)
	allGangGroupSatisfied := true
	//only the gang itself
	if len(gangGroup) == 0 {
		allGangGroupSatisfied, _ = gangCache.IsGangResourceSatisfied(gangName)
	} else {
		//check each gang group
		for _, groupName := range gangGroup {
			if satisfied, _ := gangCache.IsGangResourceSatisfied(groupName); !satisfied {
				allGangGroupSatisfied = false
				break
			}
		}
	}
	if !allGangGroupSatisfied {
		return waitTime, Wait
	}
	return 0, Success
}

// ActivateGang
//Put all the pods belong to the Gang which in UnSchedulableQueue or backoffQueue back to activeQueue,
func (p *GangPlugin) ActivateGang(pod *corev1.Pod, state *framework.CycleState) {
	gangName := pod.Annotations[GangNameAnnotation]
	pods, err := p.podLister.Pods(pod.Namespace).List(nil)
	if err != nil {
		klog.Errorf("ActivateGang Failed to list pods belong to a Gang: %v", gangName)
		return
	}
	for i := range pods {
		if pods[i].UID == pod.UID {
			pods = append(pods[:i], pods[i+1:]...)
			break
		}
	}
	if len(pods) != 0 {
		if c, err := state.Read(framework.PodsToActivateKey); err == nil {
			if s, ok := c.(*framework.PodsToActivate); ok {
				s.Lock()
				for _, pod := range pods {
					namespacedName := GetNamespacedName(pod)
					s.Map[namespacedName] = pod
				}
				s.Unlock()
			}
		}
	}
}

func (p *GangPlugin) AllowGangGroup(pod *corev1.Pod) {
	gangName := pod.Annotations[GangNameAnnotation]
	gangCache := p.gangCache
	gangGroup, _ := gangCache.GetGangGroup(gangName)
	//allow only the gang itself
	if len(gangGroup) == 0 {
		p.frameworkHandler.IterateOverWaitingPods(func(waitingPod framework.WaitingPod) {
			if waitingPod.GetPod().Annotations[GangNameAnnotation] == gangName {
				klog.Infof("Permit allows pod %v from gang %v", waitingPod.GetPod().Name, gangName)
				waitingPod.Allow(p.Name())
			}
		})
	} else {
		//allow each gang group
		for _, groupName := range gangGroup {
			p.frameworkHandler.IterateOverWaitingPods(func(waitingPod framework.WaitingPod) {
				if waitingPod.GetPod().Annotations[GangNameAnnotation] == groupName {
					klog.Infof("Permit allows pod %v from gang %v", waitingPod.GetPod().Name, gangName)
					waitingPod.Allow(p.Name())
				}
			})
		}
	}
	klog.Infof("Permit allows pod %v from gang %v", pod.Name, gangName)
}

// GetNamespacedName returns the namespaced name.
func GetNamespacedName(obj metav1.Object) string {
	return fmt.Sprintf("%v/%v", obj.GetNamespace(), obj.GetName())
}

func (p *GangPlugin) GetGangCreatTime(pod *corev1.Pod) time.Time {
	gangName := pod.Annotations[GangNameAnnotation]
	gangCache := p.gangCache
	createTime, found := gangCache.GetCreateTime(gangName)
	if !found {
		klog.Infof("GetGangCreatTime: gang %v is not found in the cache", gangName)
		return time.Time{}
	}
	return createTime
}
