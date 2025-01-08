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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

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
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

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

	if sg.Spec.GloballyEnabled.Running || sg.Spec.GloballyEnabled.Staging {
		sg.Labels = tools.SetMapValue(sg.Labels, korifiv1alpha1.CFSecurityGroupTypeLabel, korifiv1alpha1.CFSecurityGroupTypeGlobal)
		if err := r.reconcileGlobalNetworkPolicies(ctx, sg); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.cleanOrphanedPolicies(ctx, sg); err != nil {
		return ctrl.Result{}, err
	}

	log.V(1).Info("CFSecurityGroup reconciled")
	return ctrl.Result{}, nil
}

func (r *Reconciler) finalizeCFSecurityGroup(ctx context.Context, sg *korifiv1alpha1.CFSecurityGroup) (ctrl.Result, error) {
	log := logr.FromContextOrDiscard(ctx).WithName("finalize-security-group")

	policies, err := r.privilegedK8sClient.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", sg.Name),
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list NetworkPolicies: %w", err)
	}

	for _, policy := range policies.Items {
		if err := r.k8sClient.Delete(ctx, &policy); err != nil {
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
	log := r.log.WithValues("securityGroup", sg.Name, "space", space)

	policy := &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: space,
			Name:      sg.Name,
		},
	}

	err := r.k8sClient.Get(ctx, client.ObjectKeyFromObject(policy), policy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.createNetworkPolicy(ctx, sg, space, workloads)
		}
		return fmt.Errorf("failed to get NetworkPolicy %s/%s: %w", space, sg.Name, err)
	}

	updatedPolicy, err := securityGroupToNetworkPolicy(sg, space, workloads)
	if err != nil {
		return err
	}

	updatedPolicy.Labels = tools.SetMapValue(updatedPolicy.Labels, korifiv1alpha1.CFSecurityGroupTypeLabel, sg.Labels[korifiv1alpha1.CFSecurityGroupTypeLabel])
	if err := r.k8sClient.Update(ctx, updatedPolicy); err != nil {
		log.Error(err, "failed to update NetworkPolicy")
		return err
	}

	return nil
}

func (r *Reconciler) cleanOrphanedPolicies(ctx context.Context, sg *korifiv1alpha1.CFSecurityGroup) error {
	policies, err := r.privilegedK8sClient.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,%s=%s",
			korifiv1alpha1.CFSecurityGroupNameLabel, sg.Name,
			korifiv1alpha1.CFSecurityGroupTypeLabel, korifiv1alpha1.CFSecurityGroupTypeSpaceScoped),
	})
	if err != nil {
		return fmt.Errorf("failed to list NetworkPolicies: %w", err)
	}

	for _, policy := range policies.Items {
		if _, exists := sg.Spec.Spaces[policy.Namespace]; !exists {
			if err := r.k8sClient.Delete(ctx, &policy); err != nil && !apierrors.IsNotFound(err) {
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
	if err := r.k8sClient.Create(ctx, policy); err != nil {
		r.log.Error(err, "failed to create NetworkPolicy", "namespace", space, "name", sg.Name)
		return err
	}

	return nil
}

func securityGroupToNetworkPolicy(sg *korifiv1alpha1.CFSecurityGroup, space string, workloads korifiv1alpha1.SecurityGroupWorkloads) (*v1.NetworkPolicy, error) {
	egressRules, err := buildEgressRules(sg.Spec.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to build egress rules: %w", err)
	}

	policy := &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sg.Name,
			Namespace: space,
			Labels: map[string]string{
				korifiv1alpha1.CFSecurityGroupNameLabel: sg.Name,
			},
		},
		Spec: v1.NetworkPolicySpec{
			PolicyTypes: []v1.PolicyType{v1.PolicyTypeEgress},
			PodSelector: metav1.LabelSelector{},
			Egress:      egressRules,
		},
	}

	var workloadTypes []string
	if workloads.Running {
		workloadTypes = append(workloadTypes, korifiv1alpha1.CFWorkloadTypeApp)
	}

	if workloads.Staging {
		workloadTypes = append(workloadTypes, korifiv1alpha1.CFWorkloadTypeBuild)
	}

	if len(workloadTypes) > 0 {
		policy.Spec.PodSelector = metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      korifiv1alpha1.CFWorkloadTypeLabelkey,
					Operator: metav1.LabelSelectorOpIn,
					Values:   workloadTypes,
				},
			},
		}
	}

	return policy, nil
}

func buildEgressRules(rules []korifiv1alpha1.SecurityGroupRule) ([]v1.NetworkPolicyEgressRule, error) {
	var egressRules []v1.NetworkPolicyEgressRule

	for _, rule := range rules {
		ports, err := buildNetworkPolicyPorts(rule)
		if err != nil {
			return nil, err
		}

		peers, err := buildNetworkPolicyPeer(rule)
		if err != nil {
			return nil, err
		}

		egressRules = append(egressRules, v1.NetworkPolicyEgressRule{
			Ports: ports,
			To:    peers,
		})
	}

	return egressRules, nil
}

func buildNetworkPolicyPorts(rule korifiv1alpha1.SecurityGroupRule) ([]v1.NetworkPolicyPort, error) {
	var ports []v1.NetworkPolicyPort

	if rule.Protocol == korifiv1alpha1.ProtocolALL {
		ports = append(ports,
			v1.NetworkPolicyPort{Protocol: tools.PtrTo(corev1.ProtocolTCP)},
			v1.NetworkPolicyPort{Protocol: tools.PtrTo(corev1.ProtocolUDP)},
		)

		return ports, nil
	}

	if strings.Contains(rule.Ports, "-") {
		port, err := parseRangePorts(rule.Ports, rule.Protocol)
		if err != nil {
			return nil, err
		}

		return []v1.NetworkPolicyPort{port}, nil
	}

	for _, portStr := range strings.Split(rule.Ports, ",") {
		port, err := portStringToInt(portStr)
		if err != nil {
			return nil, err
		}

		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("port %d out of valid range (1-65535)", port)
		}

		ports = append(ports, v1.NetworkPolicyPort{
			Protocol: getProtocol(rule.Protocol),
			Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: int32(port)},
		})
	}

	return ports, nil
}

func parseRangePorts(ports, protocol string) (v1.NetworkPolicyPort, error) {
	rangePorts := strings.Split(ports, "-")
	if len(rangePorts) != 2 {
		return v1.NetworkPolicyPort{}, fmt.Errorf("invalid port range format: %s", ports)
	}

	start, err := portStringToInt(rangePorts[0])
	if err != nil {
		return v1.NetworkPolicyPort{}, err
	}

	end, err := portStringToInt(rangePorts[1])
	if err != nil {
		return v1.NetworkPolicyPort{}, err
	}
	//TODO: Check if needed
	if start < 1 || end > 65535 || start > end {
		return v1.NetworkPolicyPort{}, fmt.Errorf("invalid port range %d-%d (must be 1-65535 and start <= end)", start, end)
	}

	return v1.NetworkPolicyPort{
		Protocol: getProtocol(protocol),
		Port:     &intstr.IntOrString{Type: intstr.Int, IntVal: int32(start)},
		EndPort:  tools.PtrTo(int32(end)),
	}, nil
}

func portStringToInt(port string) (int, error) {
	p, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil {
		return 0, fmt.Errorf("invalid port %s: %w", port, err)
	}

	return p, nil
}

func buildNetworkPolicyPeer(rule korifiv1alpha1.SecurityGroupRule) ([]v1.NetworkPolicyPeer, error) {
	var peers []v1.NetworkPolicyPeer

	if strings.Contains(rule.Destination, "-") {
		parts := strings.Split(rule.Destination, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid IP range format: %s", rule.Destination)
		}

		cidrs, err := generateCIDRs(parts[0], parts[1])
		if err != nil {
			return nil, err
		}

		for _, cidr := range cidrs {
			peers = append(peers, v1.NetworkPolicyPeer{IPBlock: &v1.IPBlock{CIDR: cidr}})
		}

		return peers, nil
	}

	if _, _, err := net.ParseCIDR(rule.Destination); err == nil {
		return []v1.NetworkPolicyPeer{{IPBlock: &v1.IPBlock{CIDR: rule.Destination}}}, nil
	}

	if ip := net.ParseIP(rule.Destination); ip != nil {
		return []v1.NetworkPolicyPeer{{IPBlock: &v1.IPBlock{CIDR: fmt.Sprintf("%s/32", rule.Destination)}}}, nil
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
func getProtocol(protocol string) *corev1.Protocol {
	switch protocol {
	case korifiv1alpha1.ProtocolTCP:
		return tools.PtrTo(corev1.ProtocolTCP)
	case korifiv1alpha1.ProtocolUDP:
		return tools.PtrTo(corev1.ProtocolUDP)
	default:
		return tools.PtrTo(corev1.ProtocolTCP) // Default to TCP for unknown protocols
	}
}
