package securitygroups

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/tools"
	"code.cloudfoundry.org/korifi/tools/k8s"
	"github.com/go-logr/logr"
	v3 "github.com/projectcalico/api/pkg/apis/projectcalico/v3"
	"github.com/projectcalico/api/pkg/client/clientset_generated/clientset"
	"github.com/projectcalico/api/pkg/lib/numorstring"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reconciler struct {
	k8sClient     client.Client
	calicoClient  clientset.Interface
	scheme        *runtime.Scheme
	log           logr.Logger
	rootNamespace string
}

func NewReconciler(
	client client.Client,
	calicoClient clientset.Interface,
	scheme *runtime.Scheme,
	log logr.Logger,
	rootNamespace string,
) *k8s.PatchingReconciler[korifiv1alpha1.CFSecurityGroup, *korifiv1alpha1.CFSecurityGroup] {
	return k8s.NewPatchingReconciler(log, client, &Reconciler{
		k8sClient:     client,
		calicoClient:  calicoClient,
		scheme:        scheme,
		log:           log,
		rootNamespace: rootNamespace,
	})
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) *builder.Builder {
	return ctrl.NewControllerManagedBy(mgr).
		For(&korifiv1alpha1.CFSecurityGroup{}).
		Named("cfsecuritygroup").
		Watches(
			&korifiv1alpha1.CFSpace{},
			handler.EnqueueRequestsFromMapFunc(r.spaceUpdatesToSecurityGroups),
		)
}

func (r *Reconciler) spaceUpdatesToSecurityGroups(ctx context.Context, o client.Object) []reconcile.Request {
	securityGroups := korifiv1alpha1.CFSecurityGroupList{}
	err := r.k8sClient.List(ctx, &securityGroups,
		client.InNamespace(r.rootNamespace),
		client.MatchingLabels{korifiv1alpha1.CFSecurityGroupTypeLabel: korifiv1alpha1.CFSecurityGroupTypeGlobal},
	)
	if err != nil {
		return []reconcile.Request{}
	}

	var requests []reconcile.Request
	for _, sg := range securityGroups.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      sg.Name,
				Namespace: sg.Namespace,
			},
		})
	}

	return requests
}

// +kubebuilder:rbac:groups=korifi.cloudfoundry.org,resources=cfsecuritygroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=korifi.cloudfoundry.org,resources=cfsecuritygroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=korifi.cloudfoundry.org,resources=cfsecuritygroups/finalizers,verbs=update
// +kubebuilder:rbac:groups=crd.projectcalico.org,resources=networkpolicies,tiers;tier.networkpolicies,verbs=get;list;watch;create;update;patch;delete

func (r *Reconciler) ReconcileResource(ctx context.Context, sg *korifiv1alpha1.CFSecurityGroup) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx)

	sg.Status.ObservedGeneration = sg.Generation
	log.V(1).Info("set observed generation", "generation", sg.Status.ObservedGeneration)

	if !sg.GetDeletionTimestamp().IsZero() {
		return r.finalizeCFSecurityGroup(ctx, sg)
	}

	if len(sg.Spec.Spaces) > 0 {
		sg.Labels = tools.SetMapValue(sg.Labels, korifiv1alpha1.CFSecurityGroupTypeLabel, korifiv1alpha1.CFSecurityGroupTypeSpaceScoped)
		if err := r.reconcileNetworkPolicies(ctx, sg); err != nil {
			return ctrl.Result{}, err
		}
	}

	// if sg.Spec.GloballyEnabled.Running || sg.Spec.GloballyEnabled.Staging {
	// 	sg.Labels = tools.SetMapValue(sg.Labels, korifiv1alpha1.CFSecurityGroupTypeLabel, korifiv1alpha1.CFSecurityGroupTypeGlobal)
	// 	if err := r.reconcileGlobalNetworkPolicies(ctx, sg); err != nil {
	// 		return ctrl.Result{}, err
	// 	}
	// }

	if err := r.cleanOrphanedPolicies(ctx, sg); err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("CFSecurityGroup reconciled")
	return ctrl.Result{}, nil
}

func (r *Reconciler) finalizeCFSecurityGroup(ctx context.Context, sg *korifiv1alpha1.CFSecurityGroup) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx).WithName("finalize-security-group")

	policies, err := r.calicoClient.ProjectcalicoV3().NetworkPolicies(sg.Name).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", sg.Name),
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list NetworkPolicies: %w", err)
	}

	for _, policy := range policies.Items {
		if err := r.calicoClient.ProjectcalicoV3().NetworkPolicies("").Delete(ctx, policy.Name, metav1.DeleteOptions{}); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete NetworkPolicy %s: %w", policy.Name, err)
		}
	}

	if controllerutil.RemoveFinalizer(sg, korifiv1alpha1.CFSecurityGroupFinalizerName) {
		log.V(1).Info("finalizer removed")
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileGlobalNetworkPolicies(ctx context.Context, sg *korifiv1alpha1.CFSecurityGroup) error {
	spaces := &korifiv1alpha1.CFSpaceList{}
	if err := r.k8sClient.List(ctx, spaces); err != nil {
		return fmt.Errorf("failed to list CFSpaces: %w", err)
	}

	for _, space := range spaces.Items {
		if err := r.reconcileNetworkPolicyForSpace(ctx, sg, space.Name,
			korifiv1alpha1.SecurityGroupWorkloads{
				Running: sg.Spec.GloballyEnabled.Running,
				Staging: sg.Spec.GloballyEnabled.Staging,
			}); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) reconcileNetworkPolicies(ctx context.Context, sg *korifiv1alpha1.CFSecurityGroup) error {
	for space, workloads := range sg.Spec.Spaces {
		if err := r.reconcileNetworkPolicyForSpace(ctx, sg, space, workloads); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) reconcileNetworkPolicyForSpace(ctx context.Context, sg *korifiv1alpha1.CFSecurityGroup, space string, workloads korifiv1alpha1.SecurityGroupWorkloads) error {
	logs := r.log.WithValues("securityGroup", sg.Name, "space", space)

	policy := &v3.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sg.Name,
			Namespace: space,
		},
	}

	// policy, err := securityGroupToNetworkPolicy(sg, space, workloads)
	// if err != nil {
	// 	return err
	// }

	_, err = ctrl.CreateOrUpdate(ctx, r.k8sClient, policy, func())

	// _, err := r.calicoClient.ProjectcalicoV3().NetworkPolicies(space).Get(ctx, sg.Name, metav1.GetOptions{})
	// if err != nil {
	// 	logs.Error(err, "not found")
	// 	if apierrors.IsNotFound(err) {
	// 		return r.createNetworkPolicy(ctx, sg, space, workloads)
	// 	}
	// 	return fmt.Errorf("failed to get NetworkPolicy %s/%s: %w", space, sg.Name, err)
	// }

	// updatedPolicy, err := securityGroupToNetworkPolicy(sg, space, workloads)
	// if err != nil {
	// 	return err
	// }

	// updatedPolicy.Labels = tools.SetMapValue(updatedPolicy.Labels, korifiv1alpha1.CFSecurityGroupTypeLabel, sg.Labels[korifiv1alpha1.CFSecurityGroupTypeLabel])
	// if _, err := r.calicoClient.ProjectcalicoV3().NetworkPolicies(space).Update(ctx, updatedPolicy, metav1.UpdateOptions{}); err != nil {
	// 	logs.Error(err, "failed to update NetworkPolicy")
	// 	return err
	// }

	return nil
}

func (r *Reconciler) cleanOrphanedPolicies(ctx context.Context, sg *korifiv1alpha1.CFSecurityGroup) error {
	policies, err := r.calicoClient.ProjectcalicoV3().NetworkPolicies("").List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
			korifiv1alpha1.CFSecurityGroupNameLabel, sg.Name,
			korifiv1alpha1.CFSecurityGroupTypeLabel, korifiv1alpha1.CFSecurityGroupTypeSpaceScoped),
	})
	if err != nil {
		return fmt.Errorf("failed to list NetworkPolicies: %w", err)
	}

	for _, policy := range policies.Items {
		if _, exists := sg.Spec.Spaces[policy.Namespace]; !exists {
			if err := r.calicoClient.ProjectcalicoV3().NetworkPolicies("").Delete(ctx, policy.Name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete orphaned NetworkPolicy %s/%s: %w", policy.Namespace, policy.Name, err)
			}
		}
	}

	return nil
}

func (r *Reconciler) createNetworkPolicy(ctx context.Context, sg *korifiv1alpha1.CFSecurityGroup, space string, workloads korifiv1alpha1.SecurityGroupWorkloads) error {
	policy, err := securityGroupToNetworkPolicy(sg, space, workloads)
	if err != nil {
		return err
	}

	policy.Labels = tools.SetMapValue(policy.Labels, korifiv1alpha1.CFSecurityGroupTypeLabel, sg.Labels[korifiv1alpha1.CFSecurityGroupTypeLabel])
	// if _, err := r.calicoClient.ProjectcalicoV3().NetworkPolicies(space).Create(ctx, policy, metav1.CreateOptions{}); err != nil {
	// 	r.log.Error(err, "failed to create NetworkPolicy", "namespace", space, "name", sg.Name)
	// 	return err
	// }
	ctrl.CreateOrUpdate()

	if err = r.k8sClient.Create(ctx, policy); err != nil {
		r.log.Error(err, "failed to create NetworkPolicy", "namespace", space, "name", sg.Name)
		return err
	}

	return nil
}

func securityGroupToNetworkPolicy(sg *korifiv1alpha1.CFSecurityGroup, space string, workloads korifiv1alpha1.SecurityGroupWorkloads) (*v3.NetworkPolicy, error) {
	egressRules, err := buildEgressRules(sg.Spec.Rules)
	if err != nil {
		return &v3.NetworkPolicy{}, err
	}

	var workloadTypes []string
	if workloads.Running {
		workloadTypes = append(workloadTypes, korifiv1alpha1.CFWorkloadTypeApp)
	}

	if workloads.Staging {
		workloadTypes = append(workloadTypes, korifiv1alpha1.CFWorkloadTypeBuild)
	}

	var selector string
	if len(workloadTypes) > 0 {
		// Construct Calico selector string: "key in { 'value1', 'value2' }"
		values := "'" + strings.Join(workloadTypes, "', '") + "'"
		selector = fmt.Sprintf("%s in { %s }", korifiv1alpha1.CFWorkloadTypeLabelkey, values)
	}

	return &v3.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sg.Name,
			Namespace: space,
		},
		Spec: v3.NetworkPolicySpec{
			Types:    []v3.PolicyType{v3.PolicyTypeEgress},
			Egress:   egressRules,
			Selector: selector,
		},
	}, nil
}

func buildEgressRules(rules []korifiv1alpha1.SecurityGroupRule) ([]v3.Rule, error) {
	var egressRules []v3.Rule

	for _, rule := range rules {
		nets, err := buildRuleNets(rule)
		if err != nil {
			return []v3.Rule{}, err
		}

		ports, err := buildRulePorts(rule)
		if err != nil {
			return []v3.Rule{}, err
		}

		egressRules = append(egressRules, v3.Rule{
			Action:   v3.Allow,
			Protocol: &numorstring.Protocol{Type: 1, StrVal: getProtocol(rule.Protocol)},
			Destination: v3.EntityRule{
				Nets:  nets,
				Ports: ports,
			},
		})
	}

	return egressRules, nil
}

func buildRulePorts(rule korifiv1alpha1.SecurityGroupRule) ([]numorstring.Port, error) {
	var ports []numorstring.Port

	if strings.Contains(rule.Ports, "-") {
		port, err := parseRangePorts(rule.Ports)
		if err != nil {
			return nil, err
		}

		return []numorstring.Port{port}, nil
	}

	for _, portStr := range strings.Split(rule.Ports, ",") {
		port, err := portStringToUint16(portStr)
		if err != nil {
			return nil, err
		}

		ports = append(ports, numorstring.Port{MinPort: port, MaxPort: port})
	}

	return ports, nil
}

func parseRangePorts(ports string) (numorstring.Port, error) {
	rangePorts := strings.Split(ports, "-")
	if len(rangePorts) != 2 {
		return numorstring.Port{}, fmt.Errorf("invalid port range format: %s", ports)
	}

	start, err := portStringToUint16(rangePorts[0])
	if err != nil {
		return numorstring.Port{}, err
	}

	end, err := portStringToUint16(rangePorts[1])
	if err != nil {
		return numorstring.Port{}, err
	}

	return numorstring.Port{
		MinPort: start,
		MaxPort: end,
	}, nil
}

func portStringToUint16(port string) (uint16, error) {
	pStr := strings.TrimSpace(port)
	if pStr == "" {
		return 0, fmt.Errorf("port value cannot be empty")
	}
	p, err := strconv.ParseUint(pStr, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid port %s: %w", pStr, err)
	}
	return uint16(p), nil
}

func buildRuleNets(rule korifiv1alpha1.SecurityGroupRule) ([]string, error) {
	if strings.Contains(rule.Destination, "-") {
		var nets []string
		parts := strings.Split(rule.Destination, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid IP range format: %s", rule.Destination)
		}

		cidrs, err := generateCIDRs(parts[0], parts[1])
		if err != nil {
			return nil, err
		}

		for _, cidr := range cidrs {
			nets = append(nets, cidr)
		}

		return nets, nil
	}

	if _, _, err := net.ParseCIDR(rule.Destination); err == nil {
		return []string{rule.Destination}, nil
	}

	if ip := net.ParseIP(rule.Destination); ip != nil {
		return []string{fmt.Sprintf("%s/32", rule.Destination)}, nil
	}

	return nil, fmt.Errorf("invalid destination: %s", rule.Destination)
}

func generateCIDRs(startIP, endIP string) ([]string, error) {
	start, err := ipToUint32(startIP)
	if err != nil {
		return nil, err
	}

	end, err := ipToUint32(endIP)
	if err != nil {
		return nil, err
	}

	if start > end {
		return nil, fmt.Errorf("start IP %s must be less than or equal to end IP %s", startIP, endIP)
	}

	var cidrs []string
	for end >= start {
		mask := uint32(0xFFFFFFFF)
		length := 32

		for mask > 0 {
			nextMask := mask << 1
			if (start&nextMask) != start || (start|^nextMask) > end {
				break
			}
			mask = nextMask
			length--
		}

		cidrs = append(cidrs, fmt.Sprintf("%s/%d", uint32ToIP(start), length))

		start |= ^mask
		if start+1 < start { // Handle overflow
			break
		}
		start++
	}

	return cidrs, nil
}

// ipToUint32 converts an IPv4 address to a uint32.
func ipToUint32(ip string) (uint32, error) {
	parsedIP := net.ParseIP(ip).To4()
	if parsedIP == nil {
		return 0, fmt.Errorf("invalid IPv4 address: %s", ip)
	}
	return uint32(parsedIP[0])<<24 | uint32(parsedIP[1])<<16 | uint32(parsedIP[2])<<8 | uint32(parsedIP[3]), nil
}

// uint32ToIP converts a uint32 to an IPv4 address string.
func uint32ToIP(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ip>>24)&0xFF, (ip>>16)&0xFF, (ip>>8)&0xFF, ip&0xFF)
}

// getProtocol maps a protocol string to a corev1.Protocol.
func getProtocol(protocol string) string {
	switch protocol {
	case korifiv1alpha1.ProtocolTCP:
		return numorstring.ProtocolTCP
	case korifiv1alpha1.ProtocolUDP:
		return numorstring.ProtocolUDP
	default:
		return numorstring.ProtocolTCP
	}
}
