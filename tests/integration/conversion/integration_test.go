/*
Copyright 2016 The Kubernetes Authors.

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

package main

import (
	"bytes"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/pkg/apis/kops/v1alpha1"
	"k8s.io/kops/pkg/apis/kops/v1alpha2"
	"k8s.io/kops/pkg/diff"
	"path"
	"strings"
	"testing"

	_ "k8s.io/kops/pkg/apis/kops/install"
)

// TestMinimal runs the test on a minimum configuration, similar to kops create cluster minimal.example.com --zones us-west-1a
func ConversionTestMinimal(t *testing.T) {
	runTest(t, "minimal", "v1alpha1", "v1alpha2")
	runTest(t, "minimal", "v1alpha2", "v1alpha1")

	runTest(t, "minimal", "v1alpha0", "v1alpha1")
	runTest(t, "minimal", "v1alpha0", "v1alpha2")
}

func runTest(t *testing.T, srcDir string, fromVersion string, toVersion string) {
	sourcePath := path.Join(srcDir, fromVersion+".yaml")
	sourceBytes, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("unexpected error reading sourcePath %q: %v", sourcePath, err)
	}

	expectedPath := path.Join(srcDir, toVersion+".yaml")
	expectedBytes, err := ioutil.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("unexpected error reading expectedPath %q: %v", expectedPath, err)
	}

	codec := kops.Codecs.UniversalDecoder(kops.SchemeGroupVersion)

	defaults := &schema.GroupVersionKind{
		Group:   v1alpha1.SchemeGroupVersion.Group,
		Version: v1alpha1.SchemeGroupVersion.Version,
	}

	yaml, ok := runtime.SerializerInfoForMediaType(kops.Codecs.SupportedMediaTypes(), "application/yaml")
	if !ok {
		t.Fatalf("no YAML serializer registered")
	}
	var encoder runtime.Encoder

	switch toVersion {
	case "v1alpha1":
		encoder = kops.Codecs.EncoderForVersion(yaml.Serializer, v1alpha1.SchemeGroupVersion)
	case "v1alpha2":
		encoder = kops.Codecs.EncoderForVersion(yaml.Serializer, v1alpha2.SchemeGroupVersion)

	default:
		t.Fatalf("unknown version %q", toVersion)
	}

	//decoder := k8sapi.Codecs.DecoderToVersion(yaml.Serializer, kops.SchemeGroupVersion)

	var actual []string

	for _, s := range strings.Split(string(sourceBytes), "\n---\n") {
		o, gvk, err := codec.Decode([]byte(s), defaults, nil)
		if err != nil {
			t.Fatalf("error parsing file %q: %v", sourcePath, err)
		}

		expectVersion := fromVersion
		if expectVersion == "v1alpha0" {
			// Our version before we had v1alpha1
			expectVersion = "v1alpha1"
		}
		if gvk.Version != expectVersion {
			t.Fatalf("unexpected version: %q vs %q", gvk.Version, expectVersion)
		}

		var b bytes.Buffer
		if err := encoder.Encode(o, &b); err != nil {
			t.Fatalf("error encoding object: %v", err)
		}

		actual = append(actual, b.String())
	}

	actualString := strings.TrimSpace(strings.Join(actual, "\n---\n\n"))
	expectedString := strings.TrimSpace(string(expectedBytes))

	if actualString != expectedString {
		diffString := diff.FormatDiff(expectedString, actualString)
		t.Logf("diff:\n%s\n", diffString)

		t.Fatalf("converted output differed from expected")
	}
}
