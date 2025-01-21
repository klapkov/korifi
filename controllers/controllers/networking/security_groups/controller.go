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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sigs.k8s.io/controller-runtime/pkg/client"
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

		networkPolicy, err := securityGroupToNetworkPolicy(securityGroup, space.Name, workloadType)
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

		networkPolicy, err := securityGroupToNetworkPolicy(securityGroup, space, workloadType)
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
	networkPolicy, err := securityGroupToNetworkPolicy(securityGroup, space, workloadType)
	if err != nil {
		return err
	}

	if err = r.k8sClient.Create(ctx, networkPolicy); err != nil {
		return err
	}

	return nil
}

func securityGroupToNetworkPolicy(securityGroup *korifiv1alpha1.CFSecurityGroup, space, workloadType string) (*v1.NetworkPolicy, error) {
	egressRules, err := buildEgressRules(securityGroup.Spec.Rules)
	if err != nil {
		return &v1.NetworkPolicy{}, fmt.Errorf("failed to build egress rules: %w", err)
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

func buildEgressRules(rules []korifiv1alpha1.SecurityGroupRule) ([]v1.NetworkPolicyEgressRule, error) {
	var egressRules []v1.NetworkPolicyEgressRule

	for _, rule := range rules {
		networkPolicyPorts, err := buildNetworkPolicyPorts(rule)
		if err != nil {
			return []v1.NetworkPolicyEgressRule{}, err
		}

		networkPolicyPeers, err := buildNetworkPolicyPeer(rule)
		if err != nil {
			return []v1.NetworkPolicyEgressRule{}, err
		}

		egressRules = append(egressRules, v1.NetworkPolicyEgressRule{
			Ports: networkPolicyPorts,
			To:    networkPolicyPeers,
		})
	}

	return egressRules, nil
}

func buildNetworkPolicyPorts(rule korifiv1alpha1.SecurityGroupRule) ([]v1.NetworkPolicyPort, error) {
	var networkPolicyPorts []v1.NetworkPolicyPort

	if rule.Protocol == "all" {
		networkPolicyPorts = append(networkPolicyPorts, v1.NetworkPolicyPort{
			Protocol: tools.PtrTo(corev1.ProtocolTCP),
		})

		networkPolicyPorts = append(networkPolicyPorts, v1.NetworkPolicyPort{
			Protocol: tools.PtrTo(corev1.ProtocolUDP),
		})

		return networkPolicyPorts, nil
	}

	if strings.Contains(rule.Ports, "-") {
		port, err := parseRangePorts(rule.Ports, rule.Protocol)
		if err != nil {
			return []v1.NetworkPolicyPort{}, err

		}
		networkPolicyPorts = append(networkPolicyPorts, port)
		return networkPolicyPorts, nil
	}

	for _, port := range strings.Split(rule.Ports, ",") {
		parsedPort, err := portStringToInt(port)
		if err != nil {
			return []v1.NetworkPolicyPort{}, err

		}

		networkPolicyPorts = append(networkPolicyPorts, v1.NetworkPolicyPort{
			Protocol: getProtocol(rule.Protocol),
			Port: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: int32(parsedPort),
			},
		})
	}

	return networkPolicyPorts, nil
}

func parseRangePorts(ports, protocol string) (v1.NetworkPolicyPort, error) {
	rangePorts := strings.Split(ports, "-")

	startPort := rangePorts[0]
	endPort := rangePorts[1]

	startParsedPort, err := portStringToInt(startPort)
	if err != nil {
		return v1.NetworkPolicyPort{}, err

	}

	endParsedPort, err := portStringToInt(endPort)
	if err != nil {
		return v1.NetworkPolicyPort{}, err

	}

	return v1.NetworkPolicyPort{
		Protocol: getProtocol(protocol),
		Port: &intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: int32(startParsedPort),
		},
		EndPort: tools.PtrTo(int32(endParsedPort)),
	}, nil
}

func portStringToInt(port string) (int, error) {
	intPort, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil {
		return 0, fmt.Errorf("invalid port : %w", err)

	}

	return intPort, nil
}

func buildNetworkPolicyPeer(rule korifiv1alpha1.SecurityGroupRule) ([]v1.NetworkPolicyPeer, error) {
	var networkPolicyPeers []v1.NetworkPolicyPeer

	if strings.Contains(rule.Destination, "-") {
		// Handle destination when it is a gange like: 192.168.1.1-192.168.1.255
		parts := strings.Split(rule.Destination, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid IP range format: %s", rule.Destination)
		}

		cidrs := generateCIDRs(parts[0], parts[1])
		for _, cidr := range cidrs {
			networkPolicyPeers = append(networkPolicyPeers, v1.NetworkPolicyPeer{
				IPBlock: &v1.IPBlock{
					CIDR: cidr,
				},
			})
		}

		return networkPolicyPeers, nil
	}

	// Handle destination when it is a valid CIDR
	if _, _, err := net.ParseCIDR(rule.Destination); err == nil {
		networkPolicyPeers = append(networkPolicyPeers, v1.NetworkPolicyPeer{
			IPBlock: &v1.IPBlock{
				CIDR: rule.Destination,
			},
		})
		return networkPolicyPeers, nil
	}

	// Handle destination when it is just an IP
	if ip := net.ParseIP(rule.Destination); ip != nil {
		networkPolicyPeers = append(networkPolicyPeers, v1.NetworkPolicyPeer{
			IPBlock: &v1.IPBlock{
				CIDR: fmt.Sprintf("%s/32", rule.Destination),
			},
		})
		return networkPolicyPeers, nil
	}

	return nil, fmt.Errorf("invalid destination: %s", rule.Destination)
}

// Converts an IP address string to its 32-bit integer representation
func ipToUint32(ip string) uint32 {
	parsedIP := net.ParseIP(ip).To4()
	if parsedIP == nil {
		panic(fmt.Sprintf("Invalid IPv4 address: %s", ip))
	}
	return uint32(parsedIP[0])<<24 | uint32(parsedIP[1])<<16 | uint32(parsedIP[2])<<8 | uint32(parsedIP[3])
}

// Converts a 32-bit integer representation of an IP address back to string
func uint32ToIP(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", (ip>>24)&0xFF, (ip>>16)&0xFF, (ip>>8)&0xFF, ip&0xFF)
}

// Generate the minimal set of CIDRs to cover the IP range
func generateCIDRs(startIP, endIP string) []string {
	a1 := ipToUint32(startIP)
	a2 := ipToUint32(endIP)
	var cidrs []string

	for a2 >= a1 {
		mask := uint32(0xFFFFFFFF)
		length := 32

		// Find the largest mask that fits within the range
		for mask > 0 {
			nextMask := mask << 1
			if (a1&nextMask) != a1 || (a1|^nextMask) > a2 {
				break
			}
			mask = nextMask
			length--
		}

		cidrs = append(cidrs, fmt.Sprintf("%s/%d", uint32ToIP(a1), length))
		a1 |= ^mask
		if a1+1 < a1 { // Handle overflow
			break
		}
		a1++
	}
	return cidrs
}

func getProtocol(protocol string) *corev1.Protocol {
	switch protocol {
	case "tcp":
		return tools.PtrTo(corev1.ProtocolTCP)
	case "udp":
		return tools.PtrTo(corev1.ProtocolUDP)
	default:
		return nil
	}
}
