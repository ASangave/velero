/*
Copyright 2018 the Heptio Ark contributors.

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
// Code generated by mockery v1.0.0. DO NOT EDIT.
package mocks

import (
	backup "github.com/heptio/ark/pkg/backup"
	"github.com/heptio/ark/pkg/util/kube"
	mock "github.com/stretchr/testify/mock"

	v1 "github.com/heptio/ark/pkg/apis/ark/v1"
)

// ItemAction is an autogenerated mock type for the ItemAction type
type ItemAction struct {
	mock.Mock
}

// AppliesTo provides a mock function with given fields:
func (_m *ItemAction) AppliesTo() (backup.ResourceSelector, error) {
	ret := _m.Called()

	var r0 backup.ResourceSelector
	if rf, ok := ret.Get(0).(func() backup.ResourceSelector); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(backup.ResourceSelector)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Execute provides a mock function with given fields: item, _a1
func (_m *ItemAction) Execute(item kube.UnstructuredObject, _a1 *v1.Backup) (kube.UnstructuredObject, []backup.ResourceIdentifier, error) {
	ret := _m.Called(item, _a1)

	var r0 kube.UnstructuredObject
	if rf, ok := ret.Get(0).(func(kube.UnstructuredObject, *v1.Backup) kube.UnstructuredObject); ok {
		r0 = rf(item, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(kube.UnstructuredObject)
		}
	}

	var r1 []backup.ResourceIdentifier
	if rf, ok := ret.Get(1).(func(kube.UnstructuredObject, *v1.Backup) []backup.ResourceIdentifier); ok {
		r1 = rf(item, _a1)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]backup.ResourceIdentifier)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(kube.UnstructuredObject, *v1.Backup) error); ok {
		r2 = rf(item, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}
