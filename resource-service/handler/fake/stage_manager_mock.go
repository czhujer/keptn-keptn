// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package handler_mock

import (
	"github.com/keptn/keptn/resource-service/models"
	"sync"
)

// IStageManagerMock is a mock implementation of handler.IStageManager.
//
// 	func TestSomethingThatUsesIStageManager(t *testing.T) {
//
// 		// make and configure a mocked handler.IStageManager
// 		mockedIStageManager := &IStageManagerMock{
// 			CreateStageFunc: func(params models.CreateStageParams) error {
// 				panic("mock out the CreateStage method")
// 			},
// 			DeleteStageFunc: func(params models.DeleteStageParams) error {
// 				panic("mock out the DeleteStage method")
// 			},
// 		}
//
// 		// use mockedIStageManager in code that requires handler.IStageManager
// 		// and then make assertions.
//
// 	}
type IStageManagerMock struct {
	// CreateStageFunc mocks the CreateStage method.
	CreateStageFunc func(params models.CreateStageParams) error

	// DeleteStageFunc mocks the DeleteStage method.
	DeleteStageFunc func(params models.DeleteStageParams) error

	// calls tracks calls to the methods.
	calls struct {
		// CreateStage holds details about calls to the CreateStage method.
		CreateStage []struct {
			// Params is the params argument value.
			Params models.CreateStageParams
		}
		// DeleteStage holds details about calls to the DeleteStage method.
		DeleteStage []struct {
			// Params is the params argument value.
			Params models.DeleteStageParams
		}
	}
	lockCreateStage sync.RWMutex
	lockDeleteStage sync.RWMutex
}

// CreateStage calls CreateStageFunc.
func (mock *IStageManagerMock) CreateStage(params models.CreateStageParams) error {
	if mock.CreateStageFunc == nil {
		panic("IStageManagerMock.CreateStageFunc: method is nil but IStageManager.CreateStage was just called")
	}
	callInfo := struct {
		Params models.CreateStageParams
	}{
		Params: params,
	}
	mock.lockCreateStage.Lock()
	mock.calls.CreateStage = append(mock.calls.CreateStage, callInfo)
	mock.lockCreateStage.Unlock()
	return mock.CreateStageFunc(params)
}

// CreateStageCalls gets all the calls that were made to CreateStage.
// Check the length with:
//     len(mockedIStageManager.CreateStageCalls())
func (mock *IStageManagerMock) CreateStageCalls() []struct {
	Params models.CreateStageParams
} {
	var calls []struct {
		Params models.CreateStageParams
	}
	mock.lockCreateStage.RLock()
	calls = mock.calls.CreateStage
	mock.lockCreateStage.RUnlock()
	return calls
}

// DeleteStage calls DeleteStageFunc.
func (mock *IStageManagerMock) DeleteStage(params models.DeleteStageParams) error {
	if mock.DeleteStageFunc == nil {
		panic("IStageManagerMock.DeleteStageFunc: method is nil but IStageManager.DeleteStage was just called")
	}
	callInfo := struct {
		Params models.DeleteStageParams
	}{
		Params: params,
	}
	mock.lockDeleteStage.Lock()
	mock.calls.DeleteStage = append(mock.calls.DeleteStage, callInfo)
	mock.lockDeleteStage.Unlock()
	return mock.DeleteStageFunc(params)
}

// DeleteStageCalls gets all the calls that were made to DeleteStage.
// Check the length with:
//     len(mockedIStageManager.DeleteStageCalls())
func (mock *IStageManagerMock) DeleteStageCalls() []struct {
	Params models.DeleteStageParams
} {
	var calls []struct {
		Params models.DeleteStageParams
	}
	mock.lockDeleteStage.RLock()
	calls = mock.calls.DeleteStage
	mock.lockDeleteStage.RUnlock()
	return calls
}