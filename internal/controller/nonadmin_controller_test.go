package controller

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/go-logr/logr"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"

	oadpv1alpha1 "github.com/openshift/oadp-operator/api/v1alpha1"
	"github.com/openshift/oadp-operator/pkg/common"
)

const defaultNonAdminImage = "quay.io/konveyor/oadp-non-admin:latest"

type ReconcileNonAdminControllerScenario struct {
	namespace       string
	dpa             string
	errMessage      string
	eventWords      []string
	nonAdminEnabled bool
	deployment      *appsv1.Deployment
}

func createTestDeployment(namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nonAdminObjectName,
			Namespace: namespace,
			Labels: map[string]string{
				"test":                   "test",
				"app.kubernetes.io/name": "wrong",
				controlPlaneKey:          "super-wrong",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(2)),
			Selector: &metav1.LabelSelector{
				MatchLabels: controlPlaneLabel,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: controlPlaneLabel,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  nonAdminObjectName,
							Image: "wrong",
						},
					},
					ServiceAccountName: "wrong-one",
				},
			},
		},
	}
}

func runReconcileNonAdminControllerTest(
	scenario ReconcileNonAdminControllerScenario,
	updateTestScenario func(scenario ReconcileNonAdminControllerScenario),
	ctx context.Context,
	envVarValue string,
) {
	updateTestScenario(scenario)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: scenario.namespace,
		},
	}
	gomega.Expect(k8sClient.Create(ctx, namespace)).To(gomega.Succeed())

	dpa := &oadpv1alpha1.DataProtectionApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenario.dpa,
			Namespace: scenario.namespace,
		},
		Spec: oadpv1alpha1.DataProtectionApplicationSpec{
			Configuration: &oadpv1alpha1.ApplicationConfig{
				Velero: &oadpv1alpha1.VeleroConfig{},
			},
			NonAdmin: &oadpv1alpha1.NonAdmin{
				Enable: ptr.To(scenario.nonAdminEnabled),
			},
		},
	}
	gomega.Expect(k8sClient.Create(ctx, dpa)).To(gomega.Succeed())

	if scenario.deployment != nil {
		gomega.Expect(k8sClient.Create(ctx, scenario.deployment)).To(gomega.Succeed())
	}

	os.Setenv("RELATED_IMAGE_NON_ADMIN_CONTROLLER", envVarValue)
	event := record.NewFakeRecorder(5)
	r := &DataProtectionApplicationReconciler{
		Client:  k8sClient,
		Scheme:  testEnv.Scheme,
		Context: ctx,
		NamespacedName: types.NamespacedName{
			Name:      scenario.dpa,
			Namespace: scenario.namespace,
		},
		EventRecorder: event,
		dpa:           dpa,
	}
	result, err := r.ReconcileNonAdminController(logr.Discard())

	if len(scenario.errMessage) == 0 {
		gomega.Expect(result).To(gomega.BeTrue())
		gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	} else {
		gomega.Expect(result).To(gomega.BeFalse())
		gomega.Expect(err).To(gomega.HaveOccurred())
		gomega.Expect(err.Error()).To(gomega.ContainSubstring(scenario.errMessage))
	}

	if scenario.eventWords != nil {
		gomega.Expect(len(event.Events)).To(gomega.Equal(1))
		message := <-event.Events
		for _, word := range scenario.eventWords {
			gomega.Expect(message).To(gomega.ContainSubstring(word))
		}
	} else {
		gomega.Expect(len(event.Events)).To(gomega.Equal(0))
	}
}

var _ = ginkgo.Describe("Test ReconcileNonAdminController function", func() {
	var (
		ctx                 = context.Background()
		currentTestScenario ReconcileNonAdminControllerScenario
		updateTestScenario  = func(scenario ReconcileNonAdminControllerScenario) {
			currentTestScenario = scenario
		}
	)

	ginkgo.AfterEach(func() {
		os.Unsetenv("RELATED_IMAGE_NON_ADMIN_CONTROLLER")

		deployment := &appsv1.Deployment{}
		if k8sClient.Get(
			ctx,
			types.NamespacedName{
				Name:      nonAdminObjectName,
				Namespace: currentTestScenario.namespace,
			},
			deployment,
		) == nil {
			gomega.Expect(k8sClient.Delete(ctx, deployment)).To(gomega.Succeed())
		}

		dpa := &oadpv1alpha1.DataProtectionApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:      currentTestScenario.dpa,
				Namespace: currentTestScenario.namespace,
			},
		}
		gomega.Expect(k8sClient.Delete(ctx, dpa)).To(gomega.Succeed())

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: currentTestScenario.namespace,
			},
		}
		gomega.Expect(k8sClient.Delete(ctx, namespace)).To(gomega.Succeed())
	})

	ginkgo.DescribeTable("Reconcile is true",
		func(scenario ReconcileNonAdminControllerScenario) {
			runReconcileNonAdminControllerTest(scenario, updateTestScenario, ctx, defaultNonAdminImage)
		},
		ginkgo.Entry("Should create non admin deployment", ReconcileNonAdminControllerScenario{
			namespace:       "test-1",
			dpa:             "test-1-dpa",
			eventWords:      []string{"Normal", "NonAdminDeploymentReconciled", "created"},
			nonAdminEnabled: true,
		}),
		ginkgo.Entry("Should update non admin deployment", ReconcileNonAdminControllerScenario{
			namespace:       "test-2",
			dpa:             "test-2-dpa",
			eventWords:      []string{"Normal", "NonAdminDeploymentReconciled", "updated"},
			nonAdminEnabled: true,
			deployment:      createTestDeployment("test-2"),
		}),
		ginkgo.Entry("Should delete non admin deployment", ReconcileNonAdminControllerScenario{
			namespace:       "test-3",
			dpa:             "test-3-dpa",
			eventWords:      []string{"Normal", "NonAdminDeploymentDeleteSucceed", "deleted"},
			nonAdminEnabled: false,
			deployment:      createTestDeployment("test-3"),
		}),
		ginkgo.Entry("Should do nothing", ReconcileNonAdminControllerScenario{
			namespace:       "test-4",
			dpa:             "test-4-dpa",
			nonAdminEnabled: false,
		}),
	)

	ginkgo.DescribeTable("Reconcile is false",
		func(scenario ReconcileNonAdminControllerScenario) {
			runReconcileNonAdminControllerTest(scenario, updateTestScenario, ctx, defaultNonAdminImage)
		},
		ginkgo.Entry("Should error because non admin container was not found in Deployment", ReconcileNonAdminControllerScenario{
			namespace:       "test-error-1",
			dpa:             "test-error-1-dpa",
			errMessage:      "could not find Non admin container in Deployment",
			nonAdminEnabled: true,
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      nonAdminObjectName,
					Namespace: "test-error-1",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: controlPlaneLabel,
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: controlPlaneLabel,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "wrong",
								Image: defaultNonAdminImage,
							}},
						},
					},
				},
			},
		}),
	)
})

func TestDPAReconcilerBuildNonAdminDeployment(t *testing.T) {
	r := &DataProtectionApplicationReconciler{dpa: &oadpv1alpha1.DataProtectionApplication{
		Spec: oadpv1alpha1.DataProtectionApplicationSpec{
			NonAdmin: &oadpv1alpha1.NonAdmin{
				Enable: ptr.To(true),
			},
			Configuration: &oadpv1alpha1.ApplicationConfig{
				Velero: &oadpv1alpha1.VeleroConfig{},
			},
		},
	}}
	t.Setenv("RELATED_IMAGE_NON_ADMIN_CONTROLLER", defaultNonAdminImage)
	deployment := createTestDeployment("test-build-deployment")
	err := r.buildNonAdminDeployment(deployment)
	if err != nil {
		t.Errorf("buildNonAdminDeployment() errored out: %v", err)
	}
	labels := deployment.GetLabels()
	if labels["test"] != "test" {
		t.Errorf("Deployment label 'test' has wrong value: %v", labels["test"])
	}
	if labels["app.kubernetes.io/name"] != "deployment" {
		t.Errorf("Deployment label 'app.kubernetes.io/name' has wrong value: %v", labels["app.kubernetes.io/name"])
	}
	if labels[controlPlaneKey] != nonAdminObjectName {
		t.Errorf("Deployment label '%v' has wrong value: %v", controlPlaneKey, labels[controlPlaneKey])
	}
	if *deployment.Spec.Replicas != 1 {
		t.Errorf("Deployment has wrong number of replicas: %v", *deployment.Spec.Replicas)
	}
	if deployment.Spec.Template.Spec.ServiceAccountName != nonAdminObjectName {
		t.Errorf("Deployment has wrong ServiceAccount: %v", deployment.Spec.Template.Spec.ServiceAccountName)
	}
}

func TestEnsureRequiredLabels(t *testing.T) {
	deployment := createTestDeployment("test-ensure-label")
	ensureRequiredLabels(deployment)
	labels := deployment.GetLabels()
	if labels["test"] != "test" {
		t.Errorf("Deployment label 'test' has wrong value: %v", labels["test"])
	}
	if labels["app.kubernetes.io/name"] != "deployment" {
		t.Errorf("Deployment label 'app.kubernetes.io/name' has wrong value: %v", labels["app.kubernetes.io/name"])
	}
	if labels[controlPlaneKey] != nonAdminObjectName {
		t.Errorf("Deployment label '%v' has wrong value: %v", controlPlaneKey, labels[controlPlaneKey])
	}
}

func TestEnsureRequiredSpecs(t *testing.T) {
	deployment := createTestDeployment("test-ensure-spec")
	dpa := &oadpv1alpha1.DataProtectionApplication{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "123456789",
		},
		Spec: oadpv1alpha1.DataProtectionApplicationSpec{
			Configuration: &oadpv1alpha1.ApplicationConfig{
				Velero: &oadpv1alpha1.VeleroConfig{
					LogLevel: logrus.DebugLevel.String(),
				},
			},
			NonAdmin: &oadpv1alpha1.NonAdmin{
				Enable: ptr.To(true),
			},
			LogFormat: oadpv1alpha1.LogFormatJSON,
		},
	}
	err := ensureRequiredSpecs(deployment, dpa, defaultNonAdminImage, corev1.PullAlways)
	if err != nil {
		t.Errorf("ensureRequiredSpecs() errored out: %v", err)
	}
	if *deployment.Spec.Replicas != 1 {
		t.Errorf("Deployment has wrong number of replicas: %v", *deployment.Spec.Replicas)
	}
	if deployment.Spec.Template.Spec.ServiceAccountName != nonAdminObjectName {
		t.Errorf("Deployment has wrong ServiceAccount: %v", deployment.Spec.Template.Spec.ServiceAccountName)
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != defaultNonAdminImage {
		t.Errorf("Deployment has wrong Image: %v", deployment.Spec.Template.Spec.Containers[0].Image)
	}
	if len(deployment.Spec.Template.Annotations[dpaResourceVersionAnnotation]) == 0 {
		t.Errorf("Deployment does not have Annotation")
	}
	for _, env := range deployment.Spec.Template.Spec.Containers[0].Env {
		if env.Name == common.LogLevelEnvVar {
			// check that we get expected int value string from the level set in config
			if expectedLevel, err := logrus.ParseLevel(logrus.DebugLevel.String()); err != nil {
				t.Errorf("Unable to parse loglevel expected")
			} else {
				if env.Value != strconv.FormatUint(uint64(expectedLevel), 10) {
					t.Errorf("log level unexpected")
				}
			}
		}
		if env.Name == common.LogFormatEnvVar {
			if env.Value != string(oadpv1alpha1.LogFormatJSON) && env.Value != string(oadpv1alpha1.LogFormatText) {
				t.Errorf("log format unexpected")
			}
		}
	}

	previousDPAAnnotationValue := deployment.DeepCopy().Spec.Template.Annotations[dpaResourceVersionAnnotation]
	updatedDPA := &oadpv1alpha1.DataProtectionApplication{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "147258369",
		},
		Spec: oadpv1alpha1.DataProtectionApplicationSpec{
			NonAdmin: &oadpv1alpha1.NonAdmin{
				Enable: ptr.To(true),
			},
			Configuration: &oadpv1alpha1.ApplicationConfig{
				Velero: &oadpv1alpha1.VeleroConfig{},
			},
		},
	}
	err = ensureRequiredSpecs(deployment, updatedDPA, defaultNonAdminImage, corev1.PullAlways)
	if err != nil {
		t.Errorf("ensureRequiredSpecs() errored out: %v", err)
	}
	if previousDPAAnnotationValue != deployment.Spec.Template.Annotations[dpaResourceVersionAnnotation] {
		t.Errorf("Deployment have different Annotation")
	}
	updatedDPA = &oadpv1alpha1.DataProtectionApplication{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "987654321",
		},
		Spec: oadpv1alpha1.DataProtectionApplicationSpec{
			Configuration: &oadpv1alpha1.ApplicationConfig{
				Velero: &oadpv1alpha1.VeleroConfig{},
			},
			NonAdmin: &oadpv1alpha1.NonAdmin{
				Enable: ptr.To(true),
				EnforceBackupSpec: &v1.BackupSpec{
					SnapshotMoveData: ptr.To(false),
				},
			},
		},
	}
	err = ensureRequiredSpecs(deployment, updatedDPA, defaultNonAdminImage, corev1.PullAlways)
	if err != nil {
		t.Errorf("ensureRequiredSpecs() errored out: %v", err)
	}
	if previousDPAAnnotationValue == deployment.Spec.Template.Annotations[dpaResourceVersionAnnotation] {
		t.Errorf("Deployment does not have different Annotation")
	}
	for _, env := range deployment.Spec.Template.Spec.Containers[0].Env {
		if env.Name == common.LogLevelEnvVar {
			// check that we get expected int value string from the level set in config
			if expectedLevel, err := logrus.ParseLevel(""); err != nil {
				// we expect logrus.ParseLevel("") to err here and returns 0
				if err == nil {
					t.Error("Expected err when level is empty from logrus.ParseLevel")
				}
				// The returned expectedLevel of 0 is panic level
				if expectedLevel != logrus.PanicLevel {
					t.Errorf("unexpected logrus.ParseLevel('') return value")
				}
				// we ignore and return empty string instead, and nac deployment will handle defaulting
				if env.Value != "" {
					t.Errorf("log level unexpected")
				}
			}
		}
	}
	previousDPAAnnotationValue = deployment.DeepCopy().Spec.Template.Annotations[dpaResourceVersionAnnotation]
	updatedDPA = &oadpv1alpha1.DataProtectionApplication{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "112233445",
		},
		Spec: oadpv1alpha1.DataProtectionApplicationSpec{
			NonAdmin: &oadpv1alpha1.NonAdmin{
				Enable: ptr.To(true),
				EnforceBackupSpec: &v1.BackupSpec{
					SnapshotMoveData: ptr.To(false),
				},
				EnforceRestoreSpec: &v1.RestoreSpec{
					RestorePVs: ptr.To(true),
				},
				EnforceBSLSpec: &oadpv1alpha1.EnforceBackupStorageLocationSpec{
					Provider: "foo-provider",
				},
			},
			Configuration: &oadpv1alpha1.ApplicationConfig{
				Velero: &oadpv1alpha1.VeleroConfig{},
			},
		},
	}
	err = ensureRequiredSpecs(deployment, updatedDPA, defaultNonAdminImage, corev1.PullAlways)
	if err != nil {
		t.Errorf("ensureRequiredSpecs() errored out: %v", err)
	}
	if previousDPAAnnotationValue == deployment.Spec.Template.Annotations[dpaResourceVersionAnnotation] {
		t.Errorf("Deployment does not have different Annotation")
	}
}

func TestDPAReconcilerCheckNonAdminEnabled(t *testing.T) {
	tests := []struct {
		name   string
		result bool
		dpa    *oadpv1alpha1.DataProtectionApplication
	}{
		{
			name:   "DPA has non admin feature enable: true so return true",
			result: true,
			dpa: &oadpv1alpha1.DataProtectionApplication{
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					NonAdmin: &oadpv1alpha1.NonAdmin{
						Enable: ptr.To(true),
					},
				},
			},
		},
		{
			name:   "DPA has non admin feature enable: false so return false",
			result: false,
			dpa: &oadpv1alpha1.DataProtectionApplication{
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					NonAdmin: &oadpv1alpha1.NonAdmin{
						Enable: ptr.To(false),
					},
				},
			},
		},
		{
			name:   "DPA has empty non admin feature spec so return false",
			result: false,
			dpa: &oadpv1alpha1.DataProtectionApplication{
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					NonAdmin: &oadpv1alpha1.NonAdmin{},
				},
			},
		},
		{
			name:   "DPA has non admin feature enable: nil so return false",
			result: false,
			dpa: &oadpv1alpha1.DataProtectionApplication{
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					NonAdmin: &oadpv1alpha1.NonAdmin{
						Enable: nil,
					},
				},
			},
		},
		{
			name:   "DPA has no non admin feature",
			result: false,
			dpa:    &oadpv1alpha1.DataProtectionApplication{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := &DataProtectionApplicationReconciler{dpa: test.dpa}
			result := r.checkNonAdminEnabled()
			if result != test.result {
				t.Errorf("Results differ: got '%v' but expected '%v'", result, test.result)
			}
		})
	}
}

func TestDPAReconcilerGetNonAdminImage(t *testing.T) {
	tests := []struct {
		name  string
		image string
		env   string
		dpa   *oadpv1alpha1.DataProtectionApplication
	}{
		{
			name:  "Get non admin image from environment variable with default value",
			image: defaultNonAdminImage,
			env:   defaultNonAdminImage,
			dpa:   &oadpv1alpha1.DataProtectionApplication{},
		},
		{
			name:  "Get non admin image from environment variable with custom value",
			image: "quay.io/openshift/oadp-non-admin:latest",
			env:   "quay.io/openshift/oadp-non-admin:latest",
			dpa:   &oadpv1alpha1.DataProtectionApplication{},
		},
		{
			name:  "Get non admin image from unsupported overrides",
			image: "quay.io/konveyor/another:latest",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					UnsupportedOverrides: map[oadpv1alpha1.UnsupportedImageKey]string{
						"nonAdminControllerImageFqin": "quay.io/konveyor/another:latest",
					},
				},
			},
		},
		{
			name:  "Get non admin image from fallback",
			image: defaultNonAdminImage,
			dpa:   &oadpv1alpha1.DataProtectionApplication{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := &DataProtectionApplicationReconciler{dpa: test.dpa}
			if len(test.env) > 0 {
				t.Setenv("RELATED_IMAGE_NON_ADMIN_CONTROLLER", test.env)
			}
			image := r.getNonAdminImage()
			if image != test.image {
				t.Errorf("Images differ: got '%v' but expected '%v'", image, test.image)
			}
		})
	}
}
