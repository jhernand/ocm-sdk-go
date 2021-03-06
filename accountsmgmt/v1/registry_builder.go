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

package v1 // github.com/openshift-online/uhc-sdk-go/accountsmgmt/v1

// RegistryBuilder contains the data and logic needed to build 'registry' objects.
//
//
type RegistryBuilder struct {
	id         *string
	href       *string
	link       bool
	name       *string
	url        *string
	teamName   *string
	orgName    *string
	type_      *string
	cloudAlias *bool
}

// NewRegistry creates a new builder of 'registry' objects.
func NewRegistry() *RegistryBuilder {
	return new(RegistryBuilder)
}

// ID sets the identifier of the object.
func (b *RegistryBuilder) ID(value string) *RegistryBuilder {
	b.id = &value
	return b
}

// HREF sets the link to the object.
func (b *RegistryBuilder) HREF(value string) *RegistryBuilder {
	b.href = &value
	return b
}

// Link sets the flag that indicates if this is a link.
func (b *RegistryBuilder) Link(value bool) *RegistryBuilder {
	b.link = value
	return b
}

// Name sets the value of the 'name' attribute
// to the given value.
//
//
func (b *RegistryBuilder) Name(value string) *RegistryBuilder {
	b.name = &value
	return b
}

// URL sets the value of the 'URL' attribute
// to the given value.
//
//
func (b *RegistryBuilder) URL(value string) *RegistryBuilder {
	b.url = &value
	return b
}

// TeamName sets the value of the 'team_name' attribute
// to the given value.
//
//
func (b *RegistryBuilder) TeamName(value string) *RegistryBuilder {
	b.teamName = &value
	return b
}

// OrgName sets the value of the 'org_name' attribute
// to the given value.
//
//
func (b *RegistryBuilder) OrgName(value string) *RegistryBuilder {
	b.orgName = &value
	return b
}

// Type sets the value of the 'type' attribute
// to the given value.
//
//
func (b *RegistryBuilder) Type(value string) *RegistryBuilder {
	b.type_ = &value
	return b
}

// CloudAlias sets the value of the 'cloud_alias' attribute
// to the given value.
//
//
func (b *RegistryBuilder) CloudAlias(value bool) *RegistryBuilder {
	b.cloudAlias = &value
	return b
}

// Build creates a 'registry' object using the configuration stored in the builder.
func (b *RegistryBuilder) Build() (object *Registry, err error) {
	object = new(Registry)
	object.id = b.id
	object.href = b.href
	object.link = b.link
	if b.name != nil {
		object.name = b.name
	}
	if b.url != nil {
		object.url = b.url
	}
	if b.teamName != nil {
		object.teamName = b.teamName
	}
	if b.orgName != nil {
		object.orgName = b.orgName
	}
	if b.type_ != nil {
		object.type_ = b.type_
	}
	if b.cloudAlias != nil {
		object.cloudAlias = b.cloudAlias
	}
	return
}
