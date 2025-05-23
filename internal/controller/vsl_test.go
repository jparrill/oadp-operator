package controller

import (
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	oadpv1alpha1 "github.com/openshift/oadp-operator/api/v1alpha1"
)

func TestDPAReconciler_ValidateVolumeSnapshotLocation(t *testing.T) {
	tests := []struct {
		name    string
		dpa     *oadpv1alpha1.DataProtectionApplication
		secret  *corev1.Secret
		want    bool
		wantErr bool
	}{
		{
			name: "test no VSLs specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},

		{
			name: "test VSL with invalid provider specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL-invalid-provider",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginAWS,
								oadpv1alpha1.DefaultPluginGCP,
								oadpv1alpha1.DefaultPluginMicrosoftAzure,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: "invalid-provider",
							},
						},
					},
				},
			},
			want:    false,
			wantErr: true,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},

		// AWS tests
		{
			name: "test AWS VSL with AWS plugin missing",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL-aws-plugin-missing",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AWSProvider,
								Config: map[string]string{
									Region: "us-east-1",
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: true,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},

		{
			name: "test AWS VSL with only region specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginAWS,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AWSProvider,
								Config: map[string]string{
									Region: "us-east-1",
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test AWS VSL with no region specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AWSProvider,
							},
						},
					},
				},
			},
			want:    false,
			wantErr: true,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test AWS VSL with region and profile specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginAWS,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AWSProvider,
								Config: map[string]string{
									Region:     "us-east-1",
									AWSProfile: "test-profile",
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test AWS VSL with region specified and invalid config value",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AWSProvider,
								Config: map[string]string{
									Region:         "us-east-1",
									"invalid-test": "foo",
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: true,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},

		// GCP tests
		{
			name: "test GCP VSL with no config values",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginGCP,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: GCPProvider,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-gcp",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test GCP VSL with GCP plugin missing",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL-GCP-plugin-missing",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: GCPProvider,
							},
						},
					},
				},
			},
			want:    false,
			wantErr: true,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-gcp",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test GCP VSL with snapshotLocation specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginGCP,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: GCPProvider,
								Config: map[string]string{
									GCPSnapshotLocation: "test-location",
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-gcp",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test GCP VSL with project specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginGCP,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: GCPProvider,
								Config: map[string]string{
									GCPProject: "alt-project",
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-gcp",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test GCP VSL with invalid config value",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: GCPProvider,
								Config: map[string]string{
									"invalid-test": "foo",
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: true,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-gcp",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},

		// Azure tests
		{
			name: "test Azure VSL with no config values",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginMicrosoftAzure,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AzureProvider,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-azure",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test Azure VSL with Azure plugin missing",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL-Azure-plugin-missing",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AzureProvider,
							},
						},
					},
				},
			},
			want:    false,
			wantErr: true,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-azure",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test Azure VSL with apiTimeout specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginMicrosoftAzure,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AzureProvider,
								Config: map[string]string{
									AzureApiTimeout: "5m",
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-azure",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test Azure VSL with resourceGroup specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginMicrosoftAzure,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AzureProvider,
								Config: map[string]string{
									ResourceGroup: "test-rg",
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-azure",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test Azure VSL with subscriptionId specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginMicrosoftAzure,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AzureProvider,
								Config: map[string]string{
									AzureSubscriptionId: "test-alt-sub",
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-azure",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test Azure VSL with incremental specified",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{
							DefaultPlugins: []oadpv1alpha1.DefaultPlugin{
								oadpv1alpha1.DefaultPluginMicrosoftAzure,
							},
						},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AzureProvider,
								Config: map[string]string{
									AzureIncremental: "false",
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-azure",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
		{
			name: "test AzureVSL with invalid config value",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AzureProvider,
								Config: map[string]string{
									"invalid-test": "foo",
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: true,
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-azure",
					Namespace: "test-ns",
				},
				Data: map[string][]byte{"cloud": []byte("dummy_data")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, err := getFakeClientFromObjects(tt.dpa, tt.secret)
			if err != nil {
				t.Errorf("error in creating fake client, likely programmer error")
			}
			r := &DataProtectionApplicationReconciler{
				Client:  fakeClient,
				Scheme:  fakeClient.Scheme(),
				Log:     logr.Discard(),
				Context: newContextForTest(),
				NamespacedName: types.NamespacedName{
					Namespace: tt.dpa.Namespace,
					Name:      tt.dpa.Name,
				},
				EventRecorder: record.NewFakeRecorder(10),
				dpa:           tt.dpa,
			}
			got, err := r.ValidateVolumeSnapshotLocations()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVolumeSnapshotLocations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValidateVolumeSnapshotLocations() got %v, want %v", got, tt.want)
			}
		})
	}

}

func TestDPAReconciler_ReconcileVolumeSnapshotLocations(t *testing.T) {
	tests := []struct {
		name    string
		dpa     *oadpv1alpha1.DataProtectionApplication
		want    bool
		wantErr bool
	}{
		{
			name: "check owner references on VSL",
			dpa: &oadpv1alpha1.DataProtectionApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL",
					Namespace: "test-ns",
				},
				Spec: oadpv1alpha1.DataProtectionApplicationSpec{
					Configuration: &oadpv1alpha1.ApplicationConfig{
						Velero: &oadpv1alpha1.VeleroConfig{},
					},
					SnapshotLocations: []oadpv1alpha1.SnapshotLocation{
						{
							Velero: &velerov1.VolumeSnapshotLocationSpec{
								Provider: AWSProvider,
								Config: map[string]string{
									Region: "us-east-1",
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, err := getFakeClientFromObjects(tt.dpa)
			if err != nil {
				t.Errorf("error in creating fake client, likely programmer error")
			}
			r := &DataProtectionApplicationReconciler{
				Client:  fakeClient,
				Scheme:  fakeClient.Scheme(),
				Log:     logr.Discard(),
				Context: newContextForTest(),
				NamespacedName: types.NamespacedName{
					Namespace: tt.dpa.Namespace,
					Name:      tt.dpa.Name,
				},
				EventRecorder: record.NewFakeRecorder(10),
				dpa:           tt.dpa,
			}
			wantVSL := &velerov1.VolumeSnapshotLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-Velero-VSL-1",
					Namespace: "test-ns",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion:         oadpv1alpha1.SchemeBuilder.GroupVersion.String(),
						Kind:               "DataProtectionApplication",
						Name:               tt.dpa.Name,
						UID:                tt.dpa.UID,
						Controller:         pointer.BoolPtr(true),
						BlockOwnerDeletion: pointer.BoolPtr(true),
					}},
				},
			}
			got, err := r.ReconcileVolumeSnapshotLocations(r.Log)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileVolumeSnapshotLocations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReconcileVolumeSnapshotLocations() got = %v, want %v", got, tt.want)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileVolumeSnapshotLocations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			vsl := &velerov1.VolumeSnapshotLocation{}
			err = r.Get(r.Context, client.ObjectKey{Namespace: "test-ns", Name: "test-Velero-VSL-1"}, vsl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReconcileVolumeSnapshotLocations() error =%v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(vsl.OwnerReferences, wantVSL.OwnerReferences) {
				t.Errorf("ReconcileVolumeSnapshotLocations() expected VSL owner references to be %#v, got %#v", wantVSL.OwnerReferences, vsl.OwnerReferences)
			}
		})
	}
}
