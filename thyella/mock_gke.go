// Code generated by MockGen. DO NOT EDIT.
// Source: gke.go

// Package thyella is a generated GoMock package.
package thyella

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockKaasProvider is a mock of KaasProvider interface
type MockKaasProvider struct {
	ctrl     *gomock.Controller
	recorder *MockKaasProviderMockRecorder
}

// MockKaasProviderMockRecorder is the mock recorder for MockKaasProvider
type MockKaasProviderMockRecorder struct {
	mock *MockKaasProvider
}

// NewMockKaasProvider creates a new mock instance
func NewMockKaasProvider(ctrl *gomock.Controller) *MockKaasProvider {
	mock := &MockKaasProvider{ctrl: ctrl}
	mock.recorder = &MockKaasProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockKaasProvider) EXPECT() *MockKaasProviderMockRecorder {
	return m.recorder
}

// GetNodePool mocks base method
func (m *MockKaasProvider) GetNodePool(ctx context.Context, clusterName, poolName string, nodes []*Node) (*NodePool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodePool", ctx, clusterName, poolName, nodes)
	ret0, _ := ret[0].(*NodePool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodePool indicates an expected call of GetNodePool
func (mr *MockKaasProviderMockRecorder) GetNodePool(ctx, clusterName, poolName, nodes interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodePool", reflect.TypeOf((*MockKaasProvider)(nil).GetNodePool), ctx, clusterName, poolName, nodes)
}

// DeleteInstance mocks base method
func (m *MockKaasProvider) DeleteInstance(ctx context.Context, clusterName string, node *Node) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteInstance", ctx, clusterName, node)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteInstance indicates an expected call of DeleteInstance
func (mr *MockKaasProviderMockRecorder) DeleteInstance(ctx, clusterName, node interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteInstance", reflect.TypeOf((*MockKaasProvider)(nil).DeleteInstance), ctx, clusterName, node)
}
