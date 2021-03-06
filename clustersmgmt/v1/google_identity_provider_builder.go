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

// GoogleIdentityProviderBuilder contains the data and logic needed to build 'google_identity_provider' objects.
//
// Details for `google` identity providers.
type GoogleIdentityProviderBuilder struct {
	clientID     *string
	clientSecret *string
	hostedDomain *string
}

// NewGoogleIdentityProvider creates a new builder of 'google_identity_provider' objects.
func NewGoogleIdentityProvider() *GoogleIdentityProviderBuilder {
	return new(GoogleIdentityProviderBuilder)
}

// ClientID sets the value of the 'client_ID' attribute
// to the given value.
//
//
func (b *GoogleIdentityProviderBuilder) ClientID(value string) *GoogleIdentityProviderBuilder {
	b.clientID = &value
	return b
}

// ClientSecret sets the value of the 'client_secret' attribute
// to the given value.
//
//
func (b *GoogleIdentityProviderBuilder) ClientSecret(value string) *GoogleIdentityProviderBuilder {
	b.clientSecret = &value
	return b
}

// HostedDomain sets the value of the 'hosted_domain' attribute
// to the given value.
//
//
func (b *GoogleIdentityProviderBuilder) HostedDomain(value string) *GoogleIdentityProviderBuilder {
	b.hostedDomain = &value
	return b
}

// Build creates a 'google_identity_provider' object using the configuration stored in the builder.
func (b *GoogleIdentityProviderBuilder) Build() (object *GoogleIdentityProvider, err error) {
	object = new(GoogleIdentityProvider)
	if b.clientID != nil {
		object.clientID = b.clientID
	}
	if b.clientSecret != nil {
		object.clientSecret = b.clientSecret
	}
	if b.hostedDomain != nil {
		object.hostedDomain = b.hostedDomain
	}
	return
}
