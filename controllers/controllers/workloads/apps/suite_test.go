package apps_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/controllers/controllers/shared"
	"code.cloudfoundry.org/korifi/controllers/controllers/workloads/apps"
	"code.cloudfoundry.org/korifi/controllers/controllers/workloads/env"
	"code.cloudfoundry.org/korifi/tests/helpers"
	"code.cloudfoundry.org/korifi/tools/k8s"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	ctx             context.Context
	stopManager     context.CancelFunc
	stopClientCache context.CancelFunc
	testEnv         *envtest.Environment
	adminClient     client.Client
	k8sManager      manager.Manager
	testNamespace   string
)

func TestWorkloadsControllers(t *testing.T) {
	SetDefaultEventuallyTimeout(10 * time.Second)
	SetDefaultEventuallyPollingInterval(250 * time.Millisecond)

	RegisterFailHandler(Fail)
	RunSpecs(t, "CFApp Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true), zap.Level(zapcore.DebugLevel)))

	ctx = context.Background()

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "helm", "korifi", "controllers", "crds"),
		},
		ErrorIfCRDPathMissing: true,
	}

	_, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())

	Expect(korifiv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme.Scheme)).To(Succeed())
})

var _ = BeforeEach(func() {
	k8sManager = helpers.NewK8sManager(testEnv, filepath.Join("helm", "korifi", "controllers", "role.yaml"))
	Expect(shared.SetupIndexWithManager(k8sManager)).To(Succeed())

	adminClient, stopClientCache = helpers.NewCachedClient(testEnv.Config)

	err := apps.NewReconciler(
		k8sManager.GetClient(),
		k8sManager.GetScheme(),
		ctrl.Log.WithName("controllers").WithName("CFApp"),
		env.NewVCAPServicesEnvValueBuilder(k8sManager.GetClient()),
		env.NewVCAPApplicationEnvValueBuilder(k8sManager.GetClient(), nil),
	).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	testNamespace = uuid.NewString()
	Expect(adminClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	})).To(Succeed())

	cfOrg := &korifiv1alpha1.CFOrg{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testNamespace,
		},
		Spec: korifiv1alpha1.CFOrgSpec{
			DisplayName: uuid.NewString(),
		},
	}
	Expect(adminClient.Create(ctx, cfOrg)).To(Succeed())
	Expect(k8s.Patch(ctx, adminClient, cfOrg, func() {
		cfOrg.Status.GUID = testNamespace
	})).To(Succeed())

	cfSpace := &korifiv1alpha1.CFSpace{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testNamespace,
		},
		Spec: korifiv1alpha1.CFSpaceSpec{
			DisplayName: uuid.NewString(),
		},
	}
	Expect(adminClient.Create(ctx, cfSpace)).To(Succeed())
	Expect(k8s.Patch(ctx, adminClient, cfSpace, func() {
		cfSpace.Status.GUID = testNamespace
	})).To(Succeed())
})

var _ = JustBeforeEach(func() {
	stopManager = helpers.StartK8sManager(k8sManager)
})

var _ = AfterEach(func() {
	stopManager()
	stopClientCache()
})

var _ = AfterSuite(func() {
	Expect(testEnv.Stop()).To(Succeed())
})
