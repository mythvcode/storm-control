// Code generated by mockery v2.52.3. DO NOT EDIT.

package mocks

import (
	ebpfloader "github.com/mythvcode/storm-control/internal/ebpfloader"

	mock "github.com/stretchr/testify/mock"
)

// MockStatsLoader is an autogenerated mock type for the StatsLoader type
type MockStatsLoader struct {
	mock.Mock
}

type MockStatsLoader_Expecter struct {
	mock *mock.Mock
}

func (_m *MockStatsLoader) EXPECT() *MockStatsLoader_Expecter {
	return &MockStatsLoader_Expecter{mock: &_m.Mock}
}

// GetStatistic provides a mock function with no fields
func (_m *MockStatsLoader) GetStatistic() (ebpfloader.Statistic, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetStatistic")
	}

	var r0 ebpfloader.Statistic
	var r1 error
	if rf, ok := ret.Get(0).(func() (ebpfloader.Statistic, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() ebpfloader.Statistic); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(ebpfloader.Statistic)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockStatsLoader_GetStatistic_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetStatistic'
type MockStatsLoader_GetStatistic_Call struct {
	*mock.Call
}

// GetStatistic is a helper method to define mock.On call
func (_e *MockStatsLoader_Expecter) GetStatistic() *MockStatsLoader_GetStatistic_Call {
	return &MockStatsLoader_GetStatistic_Call{Call: _e.mock.On("GetStatistic")}
}

func (_c *MockStatsLoader_GetStatistic_Call) Run(run func()) *MockStatsLoader_GetStatistic_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStatsLoader_GetStatistic_Call) Return(_a0 ebpfloader.Statistic, _a1 error) *MockStatsLoader_GetStatistic_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStatsLoader_GetStatistic_Call) RunAndReturn(run func() (ebpfloader.Statistic, error)) *MockStatsLoader_GetStatistic_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockStatsLoader creates a new instance of MockStatsLoader. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockStatsLoader(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockStatsLoader {
	mock := &MockStatsLoader{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
