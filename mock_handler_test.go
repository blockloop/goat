// Code generated by mockery v1.0.0
package boar

import mock "github.com/stretchr/testify/mock"

// MockHandler is an autogenerated mock type for the Handler type
type MockHandler struct {
	mock.Mock
}

// Handle provides a mock function with given fields: _a0
func (_m *MockHandler) Handle(_a0 Context) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(Context) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}