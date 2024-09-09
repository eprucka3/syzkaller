// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	flatrpc "github.com/google/syzkaller/pkg/flatrpc"
	mock "github.com/stretchr/testify/mock"

	prog "github.com/google/syzkaller/prog"

	queue "github.com/google/syzkaller/pkg/fuzzer/queue"

	signal "github.com/google/syzkaller/pkg/signal"

	vminfo "github.com/google/syzkaller/pkg/vminfo"
)

// Manager is an autogenerated mock type for the Manager type
type Manager struct {
	mock.Mock
}

// BugFrames provides a mock function with given fields:
func (_m *Manager) BugFrames() ([]string, []string) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for BugFrames")
	}

	var r0 []string
	var r1 []string
	if rf, ok := ret.Get(0).(func() ([]string, []string)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func() []string); ok {
		r1 = rf()
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]string)
		}
	}

	return r0, r1
}

// CoverageFilter provides a mock function with given fields: modules
func (_m *Manager) CoverageFilter(modules []*vminfo.KernelModule) []uint64 {
	ret := _m.Called(modules)

	if len(ret) == 0 {
		panic("no return value specified for CoverageFilter")
	}

	var r0 []uint64
	if rf, ok := ret.Get(0).(func([]*vminfo.KernelModule) []uint64); ok {
		r0 = rf(modules)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]uint64)
		}
	}

	return r0
}

// MachineChecked provides a mock function with given fields: features, syscalls
func (_m *Manager) MachineChecked(features flatrpc.Feature, syscalls map[*prog.Syscall]bool) queue.Source {
	ret := _m.Called(features, syscalls)

	if len(ret) == 0 {
		panic("no return value specified for MachineChecked")
	}

	var r0 queue.Source
	if rf, ok := ret.Get(0).(func(flatrpc.Feature, map[*prog.Syscall]bool) queue.Source); ok {
		r0 = rf(features, syscalls)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(queue.Source)
		}
	}

	return r0
}

// MaxSignal provides a mock function with given fields:
func (_m *Manager) MaxSignal() signal.Signal {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for MaxSignal")
	}

	var r0 signal.Signal
	if rf, ok := ret.Get(0).(func() signal.Signal); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(signal.Signal)
		}
	}

	return r0
}

// NewManager creates a new instance of Manager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *Manager {
	mock := &Manager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}