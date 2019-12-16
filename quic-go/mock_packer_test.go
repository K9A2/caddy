// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/lucas-clemente/quic-go (interfaces: Packer)

// Package quic is a generated GoMock package.
package quic

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	handshake "github.com/caddyserver/caddy/v2/quic-go/core/handshake"
	protocol "github.com/caddyserver/caddy/v2/quic-go/core/protocol"
	wire "github.com/caddyserver/caddy/v2/quic-go/core/wire"
)

// MockPacker is a mock of Packer interface
type MockPacker struct {
	ctrl     *gomock.Controller
	recorder *MockPackerMockRecorder
}

// MockPackerMockRecorder is the mock recorder for MockPacker
type MockPackerMockRecorder struct {
	mock *MockPacker
}

// NewMockPacker creates a new mock instance
func NewMockPacker(ctrl *gomock.Controller) *MockPacker {
	mock := &MockPacker{ctrl: ctrl}
	mock.recorder = &MockPackerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPacker) EXPECT() *MockPackerMockRecorder {
	return m.recorder
}

// HandleTransportParameters mocks base method
func (m *MockPacker) HandleTransportParameters(arg0 *handshake.TransportParameters) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "HandleTransportParameters", arg0)
}

// HandleTransportParameters indicates an expected call of HandleTransportParameters
func (mr *MockPackerMockRecorder) HandleTransportParameters(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HandleTransportParameters", reflect.TypeOf((*MockPacker)(nil).HandleTransportParameters), arg0)
}

// MaybePackAckPacket mocks base method
func (m *MockPacker) MaybePackAckPacket() (*packedPacket, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MaybePackAckPacket")
	ret0, _ := ret[0].(*packedPacket)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MaybePackAckPacket indicates an expected call of MaybePackAckPacket
func (mr *MockPackerMockRecorder) MaybePackAckPacket() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MaybePackAckPacket", reflect.TypeOf((*MockPacker)(nil).MaybePackAckPacket))
}

// MaybePackProbePacket mocks base method
func (m *MockPacker) MaybePackProbePacket(arg0 protocol.EncryptionLevel) (*packedPacket, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MaybePackProbePacket", arg0)
	ret0, _ := ret[0].(*packedPacket)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MaybePackProbePacket indicates an expected call of MaybePackProbePacket
func (mr *MockPackerMockRecorder) MaybePackProbePacket(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MaybePackProbePacket", reflect.TypeOf((*MockPacker)(nil).MaybePackProbePacket), arg0)
}

// PackConnectionClose mocks base method
func (m *MockPacker) PackConnectionClose(arg0 *wire.ConnectionCloseFrame) (*packedPacket, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PackConnectionClose", arg0)
	ret0, _ := ret[0].(*packedPacket)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PackConnectionClose indicates an expected call of PackConnectionClose
func (mr *MockPackerMockRecorder) PackConnectionClose(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PackConnectionClose", reflect.TypeOf((*MockPacker)(nil).PackConnectionClose), arg0)
}

// PackPacket mocks base method
func (m *MockPacker) PackPacket() (*packedPacket, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PackPacket")
	ret0, _ := ret[0].(*packedPacket)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PackPacket indicates an expected call of PackPacket
func (mr *MockPackerMockRecorder) PackPacket() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PackPacket", reflect.TypeOf((*MockPacker)(nil).PackPacket))
}

// SetToken mocks base method
func (m *MockPacker) SetToken(arg0 []byte) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetToken", arg0)
}

// SetToken indicates an expected call of SetToken
func (mr *MockPackerMockRecorder) SetToken(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetToken", reflect.TypeOf((*MockPacker)(nil).SetToken), arg0)
}
