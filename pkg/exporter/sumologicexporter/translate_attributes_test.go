// Copyright 2021 Sumo Logic, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sumologicexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestTranslateAttributes(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertString("host.name", "testing-host")
	attributes.InsertString("host.id", "my-host-id")
	attributes.InsertString("host.type", "my-host-type")
	attributes.InsertString("k8s.cluster.name", "testing-cluster")
	attributes.InsertString("k8s.deployment.name", "my-deployment-name")
	attributes.InsertString("k8s.namespace.name", "my-namespace-name")
	attributes.InsertString("cloud.account.id", "my-account-id")
	attributes.InsertString("cloud.availability_zone", "my-zone")
	attributes.InsertString("cloud.region", "my-region")
	require.Equal(t, 9, attributes.Len())

	translateAttributes(attributes)

	assert.Equal(t, 9, attributes.Len())
	assertAttribute(t, attributes, "host", "testing-host")
	assertAttribute(t, attributes, "host.name", "")
	assertAttribute(t, attributes, "AccountId", "my-account-id")
	assertAttribute(t, attributes, "cloud.account.id", "")
	assertAttribute(t, attributes, "AvailabilityZone", "my-zone")
	assertAttribute(t, attributes, "clout.availability_zone", "")
	assertAttribute(t, attributes, "Region", "my-region")
	assertAttribute(t, attributes, "cloud.region", "")
	assertAttribute(t, attributes, "InstanceId", "my-host-id")
	assertAttribute(t, attributes, "host.id", "")
	assertAttribute(t, attributes, "InstanceType", "my-host-type")
	assertAttribute(t, attributes, "host.type", "")
	assertAttribute(t, attributes, "Cluster", "testing-cluster")
	assertAttribute(t, attributes, "k8s.cluster.name", "")
	assertAttribute(t, attributes, "Deployment", "my-deployment-name")
	assertAttribute(t, attributes, "k8s.deployment.name", "")
	assertAttribute(t, attributes, "Namespace", "my-namespace-name")
	assertAttribute(t, attributes, "k8s.namespace.name", "")
}

func TestTranslateAttributesDoesNothingWhenAttributeDoesNotExist(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	require.Equal(t, 0, attributes.Len())

	translateAttributes(attributes)

	assert.Equal(t, 0, attributes.Len())
	assertAttribute(t, attributes, "host", "")
}

func TestTranslateAttributesLeavesOtherAttributesUnchanged(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertString("one", "one1")
	attributes.InsertString("host.name", "host1")
	attributes.InsertString("three", "three1")
	require.Equal(t, 3, attributes.Len())

	translateAttributes(attributes)

	assert.Equal(t, 3, attributes.Len())
	assertAttribute(t, attributes, "one", "one1")
	assertAttribute(t, attributes, "host", "host1")
	assertAttribute(t, attributes, "three", "three1")
}

func TestTranslateAttributesDoesNotOverwriteExistingAttribute(t *testing.T) {
	attributes := pdata.NewAttributeMap()
	attributes.InsertString("host", "host1")
	attributes.InsertString("host.name", "hostname1")
	require.Equal(t, 2, attributes.Len())

	translateAttributes(attributes)

	assert.Equal(t, 2, attributes.Len())
	assertAttribute(t, attributes, "host", "host1")
}

func TestTranslateAttributesDoesNotOverwriteMultipleExistingAttributes(t *testing.T) {
	// Note: Current implementation of pdata.AttributeMap does not allow to insert duplicate keys.
	// See https://cloud-native.slack.com/archives/C01N5UCHTEH/p1624020829067500
	attributes := pdata.NewAttributeMap()
	attributes.InsertString("host", "host1")
	attributes.InsertString("host", "host2")
	require.Equal(t, 1, attributes.Len())
	attributes.InsertString("host.name", "hostname1")
	require.Equal(t, 2, attributes.Len())

	translateAttributes(attributes)

	assert.Equal(t, 2, attributes.Len())
	assertAttribute(t, attributes, "host", "host1")
}

func assertAttribute(t *testing.T, metadata pdata.AttributeMap, attributeName string, expectedValue string) {
	value, exists := metadata.Get(attributeName)

	if expectedValue == "" {
		assert.False(t, exists)
	} else {
		assert.True(t, exists)
		assert.Equal(t, expectedValue, value.StringVal())

	}
}

func TestTranslateConfigValue(t *testing.T) {
	translatedValue := translateConfigValue("%{k8s.pod.name}-%{host.name}/%{pod}-%{host}")

	assert.Equal(t, "%{pod}-%{host}/undefined-undefined", translatedValue)
}
