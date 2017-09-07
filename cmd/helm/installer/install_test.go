/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package installer // import "k8s.io/helm/cmd/helm/installer"

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	testcore "k8s.io/client-go/testing"

	"k8s.io/helm/pkg/version"
)

func TestDeploymentManifest(t *testing.T) {
	tests := []struct {
		name            string
		image           string
		canary          bool
		expect          string
		imagePullPolicy v1.PullPolicy
	}{
		{"default", "", false, "gcr.io/kubernetes-helm/tiller:" + version.Version, "IfNotPresent"},
		{"canary", "example.com/tiller", true, "gcr.io/kubernetes-helm/tiller:canary", "Always"},
		{"custom", "example.com/tiller:latest", false, "example.com/tiller:latest", "IfNotPresent"},
	}

	for _, tt := range tests {
		o, err := DeploymentManifest(&Options{Namespace: v1.NamespaceDefault, ImageSpec: tt.image, UseCanary: tt.canary})
		if err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}
		var dep v1beta1.Deployment
		if err := yaml.Unmarshal([]byte(o), &dep); err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}

		if got := dep.Spec.Template.Spec.Containers[0].Image; got != tt.expect {
			t.Errorf("%s: expected image %q, got %q", tt.name, tt.expect, got)
		}

		if got := dep.Spec.Template.Spec.Containers[0].ImagePullPolicy; got != tt.imagePullPolicy {
			t.Errorf("%s: expected imagePullPolicy %q, got %q", tt.name, tt.imagePullPolicy, got)
		}

		if got := dep.Spec.Template.Spec.Containers[0].Env[0].Value; got != v1.NamespaceDefault {
			t.Errorf("%s: expected namespace %q, got %q", tt.name, v1.NamespaceDefault, got)
		}
	}
}

func TestDeploymentManifestForServiceAccount(t *testing.T) {
	tests := []struct {
		name            string
		image           string
		canary          bool
		expect          string
		imagePullPolicy v1.PullPolicy
		serviceAccount  string
	}{
		{"withSA", "", false, "gcr.io/kubernetes-helm/tiller:latest", "IfNotPresent", "service-account"},
		{"withoutSA", "", false, "gcr.io/kubernetes-helm/tiller:latest", "IfNotPresent", ""},
	}
	for _, tt := range tests {
		o, err := DeploymentManifest(&Options{Namespace: v1.NamespaceDefault, ImageSpec: tt.image, UseCanary: tt.canary, ServiceAccount: tt.serviceAccount})
		if err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}

		var d v1beta1.Deployment
		if err := yaml.Unmarshal([]byte(o), &d); err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}
		if got := d.Spec.Template.Spec.ServiceAccountName; got != tt.serviceAccount {
			t.Errorf("%s: expected service account value %q, got %q", tt.name, tt.serviceAccount, got)
		}
	}
}

func TestDeploymentManifest_WithTLS(t *testing.T) {
	tests := []struct {
		opts   Options
		name   string
		enable string
		verify string
	}{
		{
			Options{Namespace: v1.NamespaceDefault, EnableTLS: true, VerifyTLS: true},
			"tls enable (true), tls verify (true)",
			"1",
			"1",
		},
		{
			Options{Namespace: v1.NamespaceDefault, EnableTLS: true, VerifyTLS: false},
			"tls enable (true), tls verify (false)",
			"1",
			"",
		},
		{
			Options{Namespace: v1.NamespaceDefault, EnableTLS: false, VerifyTLS: true},
			"tls enable (false), tls verify (true)",
			"1",
			"1",
		},
	}
	for _, tt := range tests {
		o, err := DeploymentManifest(&tt.opts)
		if err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}

		var d v1beta1.Deployment
		if err := yaml.Unmarshal([]byte(o), &d); err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}
		// verify environment variable in deployment reflect the use of tls being enabled.
		if got := d.Spec.Template.Spec.Containers[0].Env[2].Value; got != tt.verify {
			t.Errorf("%s: expected tls verify env value %q, got %q", tt.name, tt.verify, got)
		}
		if got := d.Spec.Template.Spec.Containers[0].Env[3].Value; got != tt.enable {
			t.Errorf("%s: expected tls enable env value %q, got %q", tt.name, tt.enable, got)
		}
	}
}

func TestDaemonSetManifest(t *testing.T) {
	tests := []struct {
		name            string
		image           string
		canary          bool
		expect          string
		imagePullPolicy v1.PullPolicy
	}{
		{"default", "", false, "gcr.io/kubernetes-helm/tiller:" + version.Version, "IfNotPresent"},
		{"canary", "example.com/tiller", true, "gcr.io/kubernetes-helm/tiller:canary", "Always"},
		{"custom", "example.com/tiller:latest", false, "example.com/tiller:latest", "IfNotPresent"},
	}

	for _, tt := range tests {
		o, err := DaemonSetManifest(&Options{Namespace: v1.NamespaceDefault, ImageSpec: tt.image, UseCanary: tt.canary})
		if err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}
		var dep v1beta1.DaemonSet
		if err := yaml.Unmarshal([]byte(o), &dep); err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}

		if got := dep.Spec.Template.Spec.Containers[0].Image; got != tt.expect {
			t.Errorf("%s: expected image %q, got %q", tt.name, tt.expect, got)
		}

		if got := dep.Spec.Template.Spec.Containers[0].ImagePullPolicy; got != tt.imagePullPolicy {
			t.Errorf("%s: expected imagePullPolicy %q, got %q", tt.name, tt.imagePullPolicy, got)
		}

		if got := dep.Spec.Template.Spec.Containers[0].Env[0].Value; got != v1.NamespaceDefault {
			t.Errorf("%s: expected namespace %q, got %q", tt.name, v1.NamespaceDefault, got)
		}
	}
}

func TestDaemonSetManifestForServiceAccount(t *testing.T) {
	tests := []struct {
		name            string
		image           string
		canary          bool
		expect          string
		imagePullPolicy v1.PullPolicy
		serviceAccount  string
	}{
		{"withSA", "", false, "gcr.io/kubernetes-helm/tiller:latest", "IfNotPresent", "service-account"},
		{"withoutSA", "", false, "gcr.io/kubernetes-helm/tiller:latest", "IfNotPresent", ""},
	}
	for _, tt := range tests {
		o, err := DaemonSetManifest(&Options{Namespace: v1.NamespaceDefault, ImageSpec: tt.image, UseCanary: tt.canary, ServiceAccount: tt.serviceAccount})
		if err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}

		var d v1beta1.DaemonSet
		if err := yaml.Unmarshal([]byte(o), &d); err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}
		if got := d.Spec.Template.Spec.ServiceAccountName; got != tt.serviceAccount {
			t.Errorf("%s: expected service account value %q, got %q", tt.name, tt.serviceAccount, got)
		}
	}
}

func TestDaemonSetManifest_WithTLS(t *testing.T) {
	tests := []struct {
		opts   Options
		name   string
		enable string
		verify string
	}{
		{
			Options{Namespace: v1.NamespaceDefault, EnableTLS: true, VerifyTLS: true},
			"tls enable (true), tls verify (true)",
			"1",
			"1",
		},
		{
			Options{Namespace: v1.NamespaceDefault, EnableTLS: true, VerifyTLS: false},
			"tls enable (true), tls verify (false)",
			"1",
			"",
		},
		{
			Options{Namespace: v1.NamespaceDefault, EnableTLS: false, VerifyTLS: true},
			"tls enable (false), tls verify (true)",
			"1",
			"1",
		},
	}
	for _, tt := range tests {
		o, err := DaemonSetManifest(&tt.opts)
		if err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}

		var d v1beta1.DaemonSet
		if err := yaml.Unmarshal([]byte(o), &d); err != nil {
			t.Fatalf("%s: error %q", tt.name, err)
		}
		// verify environment variable in deployment reflect the use of tls being enabled.
		if got := d.Spec.Template.Spec.Containers[0].Env[2].Value; got != tt.verify {
			t.Errorf("%s: expected tls verify env value %q, got %q", tt.name, tt.verify, got)
		}
		if got := d.Spec.Template.Spec.Containers[0].Env[3].Value; got != tt.enable {
			t.Errorf("%s: expected tls enable env value %q, got %q", tt.name, tt.enable, got)
		}
	}
}

func TestServiceManifest(t *testing.T) {
	o, err := ServiceManifest(v1.NamespaceDefault)
	if err != nil {
		t.Fatalf("error %q", err)
	}
	var svc v1.Service
	if err := yaml.Unmarshal([]byte(o), &svc); err != nil {
		t.Fatalf("error %q", err)
	}

	if got := svc.ObjectMeta.Namespace; got != v1.NamespaceDefault {
		t.Errorf("expected namespace %s, got %s", v1.NamespaceDefault, got)
	}
}

func TestSecretManifest(t *testing.T) {
	o, err := SecretManifest(&Options{
		VerifyTLS:     true,
		EnableTLS:     true,
		Namespace:     v1.NamespaceDefault,
		TLSKeyFile:    tlsTestFile(t, "key.pem"),
		TLSCertFile:   tlsTestFile(t, "crt.pem"),
		TLSCaCertFile: tlsTestFile(t, "ca.pem"),
	})

	if err != nil {
		t.Fatalf("error %q", err)
	}

	var obj v1.Secret
	if err := yaml.Unmarshal([]byte(o), &obj); err != nil {
		t.Fatalf("error %q", err)
	}

	if got := obj.ObjectMeta.Namespace; got != v1.NamespaceDefault {
		t.Errorf("expected namespace %s, got %s", v1.NamespaceDefault, got)
	}
	if _, ok := obj.Data["tls.key"]; !ok {
		t.Errorf("missing 'tls.key' in generated secret object")
	}
	if _, ok := obj.Data["tls.crt"]; !ok {
		t.Errorf("missing 'tls.crt' in generated secret object")
	}
	if _, ok := obj.Data["ca.crt"]; !ok {
		t.Errorf("missing 'ca.crt' in generated secret object")
	}
}

func TestInstall(t *testing.T) {
	image := "gcr.io/kubernetes-helm/tiller:v2.0.0"

	fc := &fake.Clientset{}
	fc.AddReactor("create", "deployments", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.CreateAction).GetObject().(*v1beta1.Deployment)
		l := obj.GetLabels()
		if reflect.DeepEqual(l, map[string]string{"app": "helm"}) {
			t.Errorf("expected labels = '', got '%s'", l)
		}
		i := obj.Spec.Template.Spec.Containers[0].Image
		if i != image {
			t.Errorf("expected image = '%s', got '%s'", image, i)
		}
		return true, obj, nil
	})
	fc.AddReactor("create", "services", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.CreateAction).GetObject().(*v1.Service)
		l := obj.GetLabels()
		if reflect.DeepEqual(l, map[string]string{"app": "helm"}) {
			t.Errorf("expected labels = '', got '%s'", l)
		}
		n := obj.ObjectMeta.Namespace
		if n != v1.NamespaceDefault {
			t.Errorf("expected namespace = '%s', got '%s'", v1.NamespaceDefault, n)
		}
		return true, obj, nil
	})

	opts := &Options{Namespace: v1.NamespaceDefault, ImageSpec: image}
	if err := Install(fc, opts); err != nil {
		t.Errorf("unexpected error: %#+v", err)
	}

	if actions := fc.Actions(); len(actions) != 2 {
		t.Errorf("unexpected actions: %v, expected 2 actions got %d", actions, len(actions))
	}
}

func TestInstall_WithTLS(t *testing.T) {
	image := "gcr.io/kubernetes-helm/tiller:v2.0.0"
	name := "tiller-secret"

	fc := &fake.Clientset{}
	fc.AddReactor("create", "deployments", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.CreateAction).GetObject().(*v1beta1.Deployment)
		l := obj.GetLabels()
		if reflect.DeepEqual(l, map[string]string{"app": "helm"}) {
			t.Errorf("expected labels = '', got '%s'", l)
		}
		i := obj.Spec.Template.Spec.Containers[0].Image
		if i != image {
			t.Errorf("expected image = '%s', got '%s'", image, i)
		}
		return true, obj, nil
	})
	fc.AddReactor("create", "services", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.CreateAction).GetObject().(*v1.Service)
		l := obj.GetLabels()
		if reflect.DeepEqual(l, map[string]string{"app": "helm"}) {
			t.Errorf("expected labels = '', got '%s'", l)
		}
		n := obj.ObjectMeta.Namespace
		if n != v1.NamespaceDefault {
			t.Errorf("expected namespace = '%s', got '%s'", v1.NamespaceDefault, n)
		}
		return true, obj, nil
	})
	fc.AddReactor("create", "secrets", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.CreateAction).GetObject().(*v1.Secret)
		if l := obj.GetLabels(); reflect.DeepEqual(l, map[string]string{"app": "helm"}) {
			t.Errorf("expected labels = '', got '%s'", l)
		}
		if n := obj.ObjectMeta.Namespace; n != v1.NamespaceDefault {
			t.Errorf("expected namespace = '%s', got '%s'", v1.NamespaceDefault, n)
		}
		if s := obj.ObjectMeta.Name; s != name {
			t.Errorf("expected name = '%s', got '%s'", name, s)
		}
		if _, ok := obj.Data["tls.key"]; !ok {
			t.Errorf("missing 'tls.key' in generated secret object")
		}
		if _, ok := obj.Data["tls.crt"]; !ok {
			t.Errorf("missing 'tls.crt' in generated secret object")
		}
		if _, ok := obj.Data["ca.crt"]; !ok {
			t.Errorf("missing 'ca.crt' in generated secret object")
		}
		return true, obj, nil
	})

	opts := &Options{
		Namespace:     v1.NamespaceDefault,
		ImageSpec:     image,
		EnableTLS:     true,
		VerifyTLS:     true,
		TLSKeyFile:    tlsTestFile(t, "key.pem"),
		TLSCertFile:   tlsTestFile(t, "crt.pem"),
		TLSCaCertFile: tlsTestFile(t, "ca.pem"),
	}

	if err := Install(fc, opts); err != nil {
		t.Errorf("unexpected error: %#+v", err)
	}

	if actions := fc.Actions(); len(actions) != 3 {
		t.Errorf("unexpected actions: %v, expected 3 actions got %d", actions, len(actions))
	}
}

func TestInstall_canary(t *testing.T) {
	fc := &fake.Clientset{}
	fc.AddReactor("create", "deployments", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.CreateAction).GetObject().(*v1beta1.Deployment)
		i := obj.Spec.Template.Spec.Containers[0].Image
		if i != "gcr.io/kubernetes-helm/tiller:canary" {
			t.Errorf("expected canary image, got '%s'", i)
		}
		return true, obj, nil
	})
	fc.AddReactor("create", "services", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.CreateAction).GetObject().(*v1.Service)
		return true, obj, nil
	})

	opts := &Options{Namespace: v1.NamespaceDefault, UseCanary: true}
	if err := Install(fc, opts); err != nil {
		t.Errorf("unexpected error: %#+v", err)
	}

	if actions := fc.Actions(); len(actions) != 2 {
		t.Errorf("unexpected actions: %v, expected 2 actions got %d", actions, len(actions))
	}
}

func TestUpgrade(t *testing.T) {
	image := "gcr.io/kubernetes-helm/tiller:v2.0.0"
	serviceAccount := "newServiceAccount"
	existingDeployment := deployment(&Options{
		Namespace:      v1.NamespaceDefault,
		ImageSpec:      "imageToReplace",
		ServiceAccount: "serviceAccountToReplace",
		UseCanary:      false,
	})
	existingService := service(v1.NamespaceDefault)

	fc := &fake.Clientset{}
	fc.AddReactor("get", "deployments", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, existingDeployment, nil
	})
	fc.AddReactor("get", "daemonsets", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	fc.AddReactor("update", "deployments", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.UpdateAction).GetObject().(*v1beta1.Deployment)
		i := obj.Spec.Template.Spec.Containers[0].Image
		if i != image {
			t.Errorf("expected image = '%s', got '%s'", image, i)
		}
		sa := obj.Spec.Template.Spec.ServiceAccountName
		if sa != serviceAccount {
			t.Errorf("expected serviceAccountName = '%s', got '%s'", serviceAccount, sa)
		}
		return true, obj, nil
	})
	fc.AddReactor("get", "services", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, existingService, nil
	})

	opts := &Options{Namespace: v1.NamespaceDefault, ImageSpec: image, ServiceAccount: serviceAccount}
	if err := Upgrade(fc, opts); err != nil {
		t.Errorf("unexpected error: %#+v", err)
	}

	if actions := fc.Actions(); len(actions) != 4 {
		t.Errorf("unexpected actions: %v, expected 4 actions got %d", actions, len(actions))
	}
}

func TestUpgrade_daemonset(t *testing.T) {
	image := "gcr.io/kubernetes-helm/tiller:v2.0.0"
	serviceAccount := "newServiceAccount"
	existingDaemonSet := daemonSet(&Options{
		Namespace:      v1.NamespaceDefault,
		ImageSpec:      "imageToReplace",
		ServiceAccount: "serviceAccountToReplace",
		UseCanary:      false,
	})
	existingService := service(v1.NamespaceDefault)

	fc := &fake.Clientset{}
	fc.AddReactor("get", "deployments", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	fc.AddReactor("get", "daemonsets", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, existingDaemonSet, nil
	})
	fc.AddReactor("update", "daemonsets", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.UpdateAction).GetObject().(*v1beta1.DaemonSet)
		i := obj.Spec.Template.Spec.Containers[0].Image
		if i != image {
			t.Errorf("expected image = '%s', got '%s'", image, i)
		}
		sa := obj.Spec.Template.Spec.ServiceAccountName
		if sa != serviceAccount {
			t.Errorf("expected serviceAccountName = '%s', got '%s'", serviceAccount, sa)
		}
		return true, obj, nil
	})
	fc.AddReactor("get", "services", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, existingService, nil
	})

	opts := &Options{Namespace: v1.NamespaceDefault, ImageSpec: image, ServiceAccount: serviceAccount, DaemonSet: true}
	if err := Upgrade(fc, opts); err != nil {
		t.Errorf("unexpected error: %#+v", err)
	}

	if actions := fc.Actions(); len(actions) != 4 {
		t.Errorf("unexpected actions: %v, expected 4 actions got %d", actions, len(actions))
	}
}

func TestUpgrade_serviceNotFound(t *testing.T) {
	image := "gcr.io/kubernetes-helm/tiller:v2.0.0"

	existingDeployment := deployment(&Options{
		Namespace: v1.NamespaceDefault,
		ImageSpec: "imageToReplace",
		UseCanary: false,
	})

	fc := &fake.Clientset{}
	fc.AddReactor("get", "deployments", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, existingDeployment, nil
	})
	fc.AddReactor("get", "daemonsets", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	fc.AddReactor("update", "deployments", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.UpdateAction).GetObject().(*v1beta1.Deployment)
		i := obj.Spec.Template.Spec.Containers[0].Image
		if i != image {
			t.Errorf("expected image = '%s', got '%s'", image, i)
		}
		return true, obj, nil
	})
	fc.AddReactor("get", "services", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(v1.Resource("services"), "1")
	})
	fc.AddReactor("create", "services", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.CreateAction).GetObject().(*v1.Service)
		n := obj.ObjectMeta.Namespace
		if n != v1.NamespaceDefault {
			t.Errorf("expected namespace = '%s', got '%s'", v1.NamespaceDefault, n)
		}
		return true, obj, nil
	})

	opts := &Options{Namespace: v1.NamespaceDefault, ImageSpec: image}
	if err := Upgrade(fc, opts); err != nil {
		t.Errorf("unexpected error: %#+v", err)
	}

	if actions := fc.Actions(); len(actions) != 5 {
		t.Errorf("unexpected actions: %v, expected 5 actions got %d", actions, len(actions))
	}
}

func TestUpgrade_daemonset_serviceNotFound(t *testing.T) {
	image := "gcr.io/kubernetes-helm/tiller:v2.0.0"

	existingDaemonSet := daemonSet(&Options{
		Namespace: v1.NamespaceDefault,
		ImageSpec: "imageToReplace",
		UseCanary: false,
		DaemonSet: true,
	})

	fc := &fake.Clientset{}
	fc.AddReactor("get", "deployments", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, nil, nil
	})
	fc.AddReactor("get", "daemonsets", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, existingDaemonSet, nil
	})
	fc.AddReactor("update", "daemonsets", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.UpdateAction).GetObject().(*v1beta1.DaemonSet)
		i := obj.Spec.Template.Spec.Containers[0].Image
		if i != image {
			t.Errorf("expected image = '%s', got '%s'", image, i)
		}
		return true, obj, nil
	})
	fc.AddReactor("get", "services", func(action testcore.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(v1.Resource("services"), "1")
	})
	fc.AddReactor("create", "services", func(action testcore.Action) (bool, runtime.Object, error) {
		obj := action.(testcore.CreateAction).GetObject().(*v1.Service)
		n := obj.ObjectMeta.Namespace
		if n != v1.NamespaceDefault {
			t.Errorf("expected namespace = '%s', got '%s'", v1.NamespaceDefault, n)
		}
		return true, obj, nil
	})

	opts := &Options{Namespace: v1.NamespaceDefault, ImageSpec: image, DaemonSet: true}
	if err := Upgrade(fc, opts); err != nil {
		t.Errorf("unexpected error: %#+v", err)
	}

	if actions := fc.Actions(); len(actions) != 5 {
		t.Errorf("unexpected actions: %v, expected 5 actions got %d", actions, len(actions))
	}
}

func tlsTestFile(t *testing.T, path string) string {
	const tlsTestDir = "../../../testdata"
	path = filepath.Join(tlsTestDir, path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("tls test file %s does not exist", path)
	}
	return path
}
