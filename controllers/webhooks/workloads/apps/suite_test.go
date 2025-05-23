package apps_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	korifiv1alpha1 "code.cloudfoundry.org/korifi/controllers/api/v1alpha1"
	"code.cloudfoundry.org/korifi/controllers/coordination"
	"code.cloudfoundry.org/korifi/tests/helpers"

	"code.cloudfoundry.org/korifi/controllers/webhooks/validation"
	"code.cloudfoundry.org/korifi/controllers/webhooks/workloads/apps"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	stopManager        context.CancelFunc
	stopClientCache    context.CancelFunc
	testEnv            *envtest.Environment
	adminClient        client.Client
	adminNonSyncClient client.Client

	ctx           context.Context
	testNamespace string
)

func TestWorkloadsWebhooks(t *testing.T) {
	SetDefaultEventuallyTimeout(10 * time.Second)
	SetDefaultEventuallyPollingInterval(250 * time.Millisecond)

	RegisterFailHandler(Fail)
	RunSpecs(t, "CFApp Webhooks Integration Test Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	webhookManifestsPath := helpers.GenerateWebhookManifest(
		"code.cloudfoundry.org/korifi/controllers/webhooks/workloads/apps",
	)
	DeferCleanup(func() {
		Expect(os.RemoveAll(filepath.Dir(webhookManifestsPath))).To(Succeed())
	})
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "helm", "korifi", "controllers", "crds"),
		},
		ErrorIfCRDPathMissing: true,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{webhookManifestsPath},
		},
	}

	adminConfig, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(adminConfig).NotTo(BeNil())

	Expect(korifiv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	k8sManager := helpers.NewK8sManager(testEnv, filepath.Join("helm", "korifi", "controllers", "role.yaml"))

	adminNonSyncClient, err = client.New(testEnv.Config, client.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	adminClient, stopClientCache = helpers.NewCachedClient(testEnv.Config)

	(&apps.AppRevWebhook{}).SetupWebhookWithManager(k8sManager)

	uncachedClient := helpers.NewUncachedClient(k8sManager.GetConfig())
	appNameDuplicateValidator := validation.NewDuplicateValidator(coordination.NewNameRegistry(uncachedClient, apps.AppEntityType))
	Expect(apps.NewValidator(appNameDuplicateValidator).SetupWebhookWithManager(k8sManager)).To(Succeed())

	stopManager = helpers.StartK8sManager(k8sManager)
})

var _ = BeforeEach(func() {
	ctx = context.Background()

	testNamespace = uuid.NewString()

	Expect(adminClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	})).To(Succeed())
})

var _ = AfterSuite(func() {
	stopManager()
	stopClientCache()
	Expect(testEnv.Stop()).To(Succeed())
})
