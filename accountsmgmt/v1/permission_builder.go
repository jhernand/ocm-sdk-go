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

// PermissionBuilder contains the data and logic needed to build 'permission' objects.
//
//
type PermissionBuilder struct {
	id           *string
	href         *string
	link         bool
	action       *Action
	resourceType *string
	roleID       *string
}

// NewPermission creates a new builder of 'permission' objects.
func NewPermission() *PermissionBuilder {
	return new(PermissionBuilder)
}

// ID sets the identifier of the object.
func (b *PermissionBuilder) ID(value string) *PermissionBuilder {
	b.id = &value
	return b
}

// HREF sets the link to the object.
func (b *PermissionBuilder) HREF(value string) *PermissionBuilder {
	b.href = &value
	return b
}

// Link sets the flag that indicates if this is a link.
func (b *PermissionBuilder) Link(value bool) *PermissionBuilder {
	b.link = value
	return b
}

// Action sets the value of the 'action' attribute
// to the given value.
//
// Possible actions for a permission.
func (b *PermissionBuilder) Action(value Action) *PermissionBuilder {
	b.action = &value
	return b
}

// ResourceType sets the value of the 'resource_type' attribute
// to the given value.
//
//
func (b *PermissionBuilder) ResourceType(value string) *PermissionBuilder {
	b.resourceType = &value
	return b
}

// RoleID sets the value of the 'role_ID' attribute
// to the given value.
//
//
func (b *PermissionBuilder) RoleID(value string) *PermissionBuilder {
	b.roleID = &value
	return b
}

// Build creates a 'permission' object using the configuration stored in the builder.
func (b *PermissionBuilder) Build() (object *Permission, err error) {
	object = new(Permission)
	object.id = b.id
	object.href = b.href
	object.link = b.link
	if b.action != nil {
		object.action = b.action
	}
	if b.resourceType != nil {
		object.resourceType = b.resourceType
	}
	if b.roleID != nil {
		object.roleID = b.roleID
	}
	return
}
