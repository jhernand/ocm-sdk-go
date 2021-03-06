/*
Copyright (c) 2019 Red Hat, Inc.

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

// IMPORTANT: This file has been generated automatically, refrain from modifying it manually as all
// your changes will be lost when the file is generated again.

package v1 // github.com/openshift-online/uhc-sdk-go/clustersmgmt/v1

// CloudProviderBuilder contains the data and logic needed to build 'cloud_provider' objects.
//
// Cloud provider.
type CloudProviderBuilder struct {
	name        *string
	displayName *string
}

// NewCloudProvider creates a new builder of 'cloud_provider' objects.
func NewCloudProvider() *CloudProviderBuilder {
	return new(CloudProviderBuilder)
}

// Name sets the value of the 'name' attribute
// to the given value.
//
//
func (b *CloudProviderBuilder) Name(value string) *CloudProviderBuilder {
	b.name = &value
	return b
}

// DisplayName sets the value of the 'display_name' attribute
// to the given value.
//
//
func (b *CloudProviderBuilder) DisplayName(value string) *CloudProviderBuilder {
	b.displayName = &value
	return b
}

// Build creates a 'cloud_provider' object using the configuration stored in the builder.
func (b *CloudProviderBuilder) Build() (object *CloudProvider, err error) {
	object = new(CloudProvider)
	if b.name != nil {
		object.name = b.name
	}
	if b.displayName != nil {
		object.displayName = b.displayName
	}
	return
}
