package securitygroups

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/tools"

	"code.cloudfoundry.org/korifi/tools/k8s"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "k8s.io/client-go/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ProtocolICMP corev1.Protocol = "ICMP"

type Reconciler struct {
	k8sClient           client.Client
	privilegedK8sClient *k8sclient.Clientset
	scheme              *runtime.Scheme
	log                 logr.Logger
	rootNamespace       string
}

func NewReconciler(
	client client.Client,
	privilegedK8sClient *k8sclient.Clientset,
	scheme *runtime.Scheme,
	log logr.Logger,
	rootNamespace string,
) *k8s.PatchingReconciler[korifiv1alpha1.CFSecurityGroup, *korifiv1alpha1.CFSecurityGroup] {
	return k8s.NewPatchingReconciler(log, client, &Reconciler{
		k8sClient:           client,
		privilegedK8sClient: privilegedK8sClient,
		scheme:              scheme,
		log:                 log,
		rootNamespace:       rootNamespace,
	})
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) *builder.Builder {
	return ctrl.NewControllerManagedBy(mgr).
		For(&korifiv1alpha1.CFSecurityGroup{}).
		Named("cfsecuritygroup").
		// WithEventFilter(predicate.NewPredicateFuncs(r.isSpaceScoped)).
		Watches(
			&korifiv1alpha1.CFSpace{},
			handler.EnqueueRequestsFromMapFunc(r.spaceUpdatesToSecurityGroups),
		)
}

func (r *Reconciler) spaceUpdatesToSecurityGroups(ctx context.Context, o client.Object) []reconcile.Request {
	_ = o.(*korifiv1alpha1.CFSpace)

	securityGroups := korifiv1alpha1.CFSecurityGroupList{}
	if err := r.k8sClient.List(ctx, &securityGroups,
		client.InNamespace(r.rootNamespace),
		client.MatchingLabels{korifiv1alpha1.CFSecurityGroupTypeLabel: korifiv1alpha1.CFSecurityGroupTypeGlobal},
	); err != nil {
		return []reconcile.Request{}
	}

	requests := []reconcile.Request{}
	for _, sc := range securityGroups.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      sc.Name,
				Namespace: sc.Namespace,
			},
		})
	}

	return requests
}

// func (r *Reconciler) isSpaceScoped(object client.Object) bool {
// 	securityGroup, ok := object.(*korifiv1alpha1.CFSecurityGroup)
// 	if !ok {
// 		return true
// 	}

// 	return !securityGroup.Spec.GloballyEnabled.Running || !securityGroup.Spec.GloballyEnabled.Staging
// }

//+kubebuilder:rbac:groups=korifi.cloudfoundry.org,resources=cfsecuritygroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=korifi.cloudfoundry.org,resources=cfsecuritygroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=korifi.cloudfoundry.org,resources=cfsecuritygroups/finalizers,verbs=update
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

func (r *Reconciler) ReconcileResource(ctx context.Context, securityGroup *korifiv1alpha1.CFSecurityGroup) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx)

	log.Info("Reconciling CFSecurityGroup", "name", securityGroup.Name, "namespace", securityGroup.Namespace)

	securityGroup.Status.ObservedGeneration = securityGroup.Generation
	log.V(1).Info("set observed generation", "generation", securityGroup.Status.ObservedGeneration)

	if !securityGroup.GetDeletionTimestamp().IsZero() {
		return r.finalizeCFSecurityGroup(ctx, securityGroup)
	}

	if len(securityGroup.Spec.RunningSpaces) != 0 {
		if err := r.reconcileNetworkPolicies(ctx, securityGroup, securityGroup.Spec.RunningSpaces, korifiv1alpha1.CFWorkloadTypeApp); err != nil {
			return ctrl.Result{}, err
		}
	}

	if len(securityGroup.Spec.StagingSpaces) != 0 {
		if err := r.reconcileNetworkPolicies(ctx, securityGroup, securityGroup.Spec.StagingSpaces, korifiv1alpha1.CFWorkloadTypeBuild); err != nil {
			return ctrl.Result{}, err
		}
	}

	if securityGroup.Spec.GloballyEnabled.Running {
		if err := r.reconcileGlobalNetworkPolicies(ctx, securityGroup, korifiv1alpha1.CFWorkloadTypeApp); err != nil {
			return ctrl.Result{}, err
		}
	}

	if securityGroup.Spec.GloballyEnabled.Staging {
		if err := r.reconcileGlobalNetworkPolicies(ctx, securityGroup, korifiv1alpha1.CFWorkloadTypeBuild); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.V(1).Info("Security Group reconciled")

	return ctrl.Result{}, nil
}

func (r *Reconciler) finalizeCFSecurityGroup(ctx context.Context, securityGroup *korifiv1alpha1.CFSecurityGroup) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx).WithName("finalize-security-group")

	networkPolicies, err := r.privilegedK8sClient.
		NetworkingV1().
		NetworkPolicies("").
		List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", korifiv1alpha1.CFSecurityGroupNameLabel, securityGroup.Name)})
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, policy := range networkPolicies.Items {
		if err = r.k8sClient.Delete(ctx, &policy); err != nil {
			return ctrl.Result{}, err
		}
	}

	if controllerutil.RemoveFinalizer(securityGroup, korifiv1alpha1.CFSecurityGroupFinalizerName) {
		log.V(1).Info("finalizer removed")
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileGlobalNetworkPolicies(ctx context.Context, securityGroup *korifiv1alpha1.CFSecurityGroup, workloadType string) error {
	spaces := &korifiv1alpha1.CFSpaceList{}
	if err := r.k8sClient.List(ctx, spaces); err != nil {
		return err
	}

	for _, space := range spaces.Items {
		networkPolicy := &v1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: space.Name,
				Name:      securityGroup.Name,
			},
		}

		if err := r.k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy); err != nil {
			if apierrors.IsNotFound(err) {
				if err = r.createNetworkPolicy(ctx, securityGroup, space.Name, workloadType); err != nil {
					return err
				}

			}

			return err
		}

		securityGroup.Labels = tools.SetMapValue(securityGroup.Labels, korifiv1alpha1.CFSecurityGroupTypeLabel, korifiv1alpha1.CFSecurityGroupTypeGlobal)

		networkPolicy, err := toNetworkPolicy(securityGroup, space.Name, workloadType)
		if err != nil {
			return err
		}

		if err = r.k8sClient.Update(ctx, networkPolicy); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) reconcileNetworkPolicies(
	ctx context.Context,
	securityGroup *korifiv1alpha1.CFSecurityGroup,
	spaces []string,
	workloadType string,
) error {
	for _, space := range spaces {
		networkPolicy := &v1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: space,
				Name:      securityGroup.Name,
			},
		}

		if err := r.k8sClient.Get(ctx, client.ObjectKeyFromObject(networkPolicy), networkPolicy); err != nil {
			if apierrors.IsNotFound(err) {
				if err = r.createNetworkPolicy(ctx, securityGroup, space, workloadType); err != nil {
					return err
				}

			}

			return err
		}

		networkPolicy, err := toNetworkPolicy(securityGroup, space, workloadType)
		if err != nil {
			return err
		}

		if err = r.k8sClient.Update(ctx, networkPolicy); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) createNetworkPolicy(ctx context.Context, securityGroup *korifiv1alpha1.CFSecurityGroup, space, workloadType string) error {
	networkPolicy, err := toNetworkPolicy(securityGroup, space, workloadType)
	if err != nil {
		return err
	}

	if err = r.k8sClient.Create(ctx, networkPolicy); err != nil {
		return err
	}

	return nil
}

func toNetworkPolicy(securityGroup *korifiv1alpha1.CFSecurityGroup, space string, workloadType string) (*v1.NetworkPolicy, error) {
	var egressRules []v1.NetworkPolicyEgressRule

	for _, rule := range securityGroup.Spec.Rules {
		egressRule := v1.NetworkPolicyEgressRule{
			To: []v1.NetworkPolicyPeer{
				{
					IPBlock: &v1.IPBlock{
						CIDR: rule.Destination,
					},
				},
			},
		}

		if len(rule.Ports) != 0 {
			ports, err := toNetworkPolicyPorts(rule.Ports, rule.Protocol)

			if err != nil {
				return nil, err
			}

			egressRule.Ports = ports
		} else {
			if rule.Protocol != "any" {
				egressRule.Ports = append(egressRule.Ports, v1.NetworkPolicyPort{
					Protocol: getProtocol(rule.Protocol),
				})
			}
		}

		egressRules = append(egressRules, egressRule)

	}

	return &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      securityGroup.Name,
			Namespace: space,
			Labels: map[string]string{
				korifiv1alpha1.CFSecurityGroupNameLabel: securityGroup.Name,
			},
		},
		Spec: v1.NetworkPolicySpec{
			PolicyTypes: []v1.PolicyType{
				v1.PolicyTypeEgress,
			},
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{korifiv1alpha1.CFWorkloadTypeLabelkey: workloadType},
			},
			Egress: egressRules,
		},
	}, nil
}

func toNetworkPolicyPorts(ports, protocol string) ([]v1.NetworkPolicyPort, error) {
	var networkPolicyPorts []v1.NetworkPolicyPort

	for _, port := range strings.Split(ports, ",") {
		parsedPort, err := strconv.Atoi(strings.TrimSpace(port))
		if err != nil {
			return []v1.NetworkPolicyPort{}, fmt.Errorf("invalid port : %w", err)

		}

		policyPort := v1.NetworkPolicyPort{
			Port: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: int32(parsedPort),
			},
		}

		// If port is not specified, it defaults to tcp, so if protocol is all, we need to do more

		if protocol != "all" {
			policyPort.Protocol = getProtocol(protocol)
		}

		networkPolicyPorts = append(networkPolicyPorts, policyPort)
	}

	return networkPolicyPorts, nil
}

func getProtocol(protocol string) *corev1.Protocol {
	switch protocol {
	case "tcp":
		tcp := corev1.ProtocolTCP
		return &tcp
	case "udp":
		udp := corev1.ProtocolUDP
		return &udp
	// kubernetes returns a error that it is not allowed
	// case "icmp":
	// 	icmp := ProtocolICMP
	// 	return &icmp
	default:
		return nil
	}
}

// func isGlobal(securityGroup *korifiv1alpha1.CFSecurityGroup) bool {
// 	return securityGroup.Spec.GloballyEnabled.Running || securityGroup.Spec.GloballyEnabled.Staging
// }
