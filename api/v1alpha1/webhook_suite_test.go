//
// Copyright (c) 2012-2023 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

package v1alpha1

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	//+kubebuilder:scaffold:imports

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc
var runtimeScheme *runtime.Scheme

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Webhook Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
		},
	}

	_cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(_cfg).NotTo(BeNil())
	cfg = _cfg

	scheme := runtime.NewScheme()
	err = AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1beta1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = corev1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = rbacv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	runtimeScheme = scheme

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(_cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// start webhook server using Manager
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(_cfg, ctrl.Options{
		Scheme:             scheme,
		Host:               webhookInstallOptions.LocalServingHost,
		Port:               webhookInstallOptions.LocalServingPort,
		CertDir:            webhookInstallOptions.LocalServingCertDir,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&KubernetesImagePuller{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:webhook

	go func() {
		err = mgr.Start(ctx)
		if err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)

	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(Succeed())

	verifyWebhookPathRegistered(mgr, "/validate-che-eclipse-org-v1alpha1-kubernetesimagepuller")
}, 60)

func verifyWebhookPathRegistered(mgr ctrl.Manager, path string) {
	Expect(func() {
		mgr.GetWebhookServer().Register(path, &webhook.Admission{})
	}).Should(Panic(), "A panic is expected when registering a duplicate path, but no panic with was detected. Verify that the '%s' webhook is registered.", path)
}

var _ = Describe("Create KubernetesImagePuller resource", func() {

	var kip *KubernetesImagePuller

	BeforeEach(func() {
		kip = &KubernetesImagePuller{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-kip",
				Namespace: "default",
			},
		}
	})

	AfterEach(func() {
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kip), &KubernetesImagePuller{}); err == nil {
			Expect(k8sClient.Delete(ctx, kip)).Should(Succeed())
		}
	})

	It("Should create a KubernetesImagePuller resource", func() {
		Expect(k8sClient.Create(ctx, kip)).Should(Succeed())
	})

	It("Should not create a KubernetesImagePuller resource from a service account without daemonset create permissions", func() {
		testServiceAccount := getServiceAccount(ctx, "test-sa")
		Expect(k8sClient.Create(ctx, testServiceAccount)).Should(Succeed())

		kipEditorRole := getKipEditorRole()
		Expect(k8sClient.Create(ctx, kipEditorRole)).Should(Succeed())

		// assign the kip editor role to the service account
		kipEditorRoleBinding := getClusterRoleBinding(ctx, kipEditorRole, testServiceAccount, "kip-editor-role-binding")
		Expect(k8sClient.Create(ctx, kipEditorRoleBinding)).Should(Succeed())

		// create new client that impersonates the new service account
		newClient := getImpersonatedClient(testServiceAccount)

		// create image puller resource
		err := newClient.Create(ctx, kip)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("admission webhook \"vkubernetesimagepuller.kb.io\" denied the request: User \"system:serviceaccount:default:test-sa\" not allowed to create daemonsets"))

		// delete created resources from cluster
		Expect(k8sClient.Delete(ctx, testServiceAccount)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, kipEditorRole)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, kipEditorRoleBinding)).Should(Succeed())
	})

	It("Should create a KubernetesImagePuller resource from a service account with daemonset create permissions", func() {
		testServiceAccount := getServiceAccount(ctx, "test-sa")
		Expect(k8sClient.Create(ctx, testServiceAccount)).Should(Succeed())

		createDaemonsetRole := getDaemonsetRole()
		kipEditorRole := getKipEditorRole()

		Expect(k8sClient.Create(ctx, createDaemonsetRole)).Should(Succeed())
		Expect(k8sClient.Create(ctx, kipEditorRole)).Should(Succeed())

		// assign the kip editor role to the service account
		kipEditorRoleBinding := getClusterRoleBinding(ctx, kipEditorRole, testServiceAccount, "kip-editor-role-binding")
		Expect(k8sClient.Create(ctx, kipEditorRoleBinding)).Should(Succeed())

		// assign the create daemonset role to the service account
		createDaemonsetRoleBinding := getClusterRoleBinding(ctx, createDaemonsetRole, testServiceAccount, "create-daemonset-role-binding")
		Expect(k8sClient.Create(ctx, createDaemonsetRoleBinding)).Should(Succeed())

		// create new client that impersonates the new service account and create image puller resource
		newClient := getImpersonatedClient(testServiceAccount)

		// create image puller resource
		Expect(newClient.Create(ctx, kip)).Should(Succeed())

		// delete created resources from cluster
		Expect(k8sClient.Delete(ctx, testServiceAccount)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, createDaemonsetRole)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, kipEditorRole)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, createDaemonsetRoleBinding)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, kipEditorRoleBinding)).Should(Succeed())
	})
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func getServiceAccount(ctx context.Context, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
	}
}

func getImpersonatedClient(sa *corev1.ServiceAccount) client.Client {
	newCfg := rest.CopyConfig(cfg)
	newCfg.Impersonate = rest.ImpersonationConfig{
		UserName: fmt.Sprintf("system:serviceaccount:%s:%s", sa.Namespace, sa.Name),
	}
	newClient, err := client.New(newCfg, client.Options{Scheme: k8sClient.Scheme()})
	Expect(err).NotTo(HaveOccurred())
	Expect(newClient).NotTo(BeNil())
	return newClient
}

func getKipEditorRole() *rbacv1.ClusterRole {
	dir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())

	filename := "kubernetesimagepuller_editor_role.yaml"
	editorRoleFile, err := os.ReadFile(filepath.Join(dir, "..", "..", "config", "rbac", filename))
	Expect(err).NotTo(HaveOccurred())

	role := rbacv1.ClusterRole{}

	codecs := serializer.NewCodecFactory(runtimeScheme)
	deserializer := codecs.UniversalDeserializer()
	_, _, err = deserializer.Decode(editorRoleFile, nil, &role)
	Expect(err).NotTo(HaveOccurred())

	return &role
}

func getDaemonsetRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "create-daemonset",
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"create"},
				APIGroups: []string{"apps"},
				Resources: []string{"daemonsets"},
			},
		},
	}
}

func getClusterRoleBinding(ctx context.Context, role *rbacv1.ClusterRole, sa *corev1.ServiceAccount, roleBindingName string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     role.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
}
