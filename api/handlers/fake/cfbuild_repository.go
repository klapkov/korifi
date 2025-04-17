// Code generated by counterfeiter. DO NOT EDIT.
package fake

import (
	"context"
	"sync"

	"code.cloudfoundry.org/korifi/api/authorization"
	"code.cloudfoundry.org/korifi/api/handlers"
	"code.cloudfoundry.org/korifi/api/repositories"
)

type CFBuildRepository struct {
	CreateBuildStub        func(context.Context, authorization.Info, repositories.CreateBuildMessage) (repositories.BuildRecord, error)
	createBuildMutex       sync.RWMutex
	createBuildArgsForCall []struct {
		arg1 context.Context
		arg2 authorization.Info
		arg3 repositories.CreateBuildMessage
	}
	createBuildReturns struct {
		result1 repositories.BuildRecord
		result2 error
	}
	createBuildReturnsOnCall map[int]struct {
		result1 repositories.BuildRecord
		result2 error
	}
	GetBuildStub        func(context.Context, authorization.Info, string) (repositories.BuildRecord, error)
	getBuildMutex       sync.RWMutex
	getBuildArgsForCall []struct {
		arg1 context.Context
		arg2 authorization.Info
		arg3 string
	}
	getBuildReturns struct {
		result1 repositories.BuildRecord
		result2 error
	}
	getBuildReturnsOnCall map[int]struct {
		result1 repositories.BuildRecord
		result2 error
	}
	GetLatestBuildByAppGUIDStub        func(context.Context, authorization.Info, string, string) (repositories.BuildRecord, error)
	getLatestBuildByAppGUIDMutex       sync.RWMutex
	getLatestBuildByAppGUIDArgsForCall []struct {
		arg1 context.Context
		arg2 authorization.Info
		arg3 string
		arg4 string
	}
	getLatestBuildByAppGUIDReturns struct {
		result1 repositories.BuildRecord
		result2 error
	}
	getLatestBuildByAppGUIDReturnsOnCall map[int]struct {
		result1 repositories.BuildRecord
		result2 error
	}
	ListBuildsStub        func(context.Context, authorization.Info) ([]repositories.BuildRecord, error)
	listBuildsMutex       sync.RWMutex
	listBuildsArgsForCall []struct {
		arg1 context.Context
		arg2 authorization.Info
	}
	listBuildsReturns struct {
		result1 []repositories.BuildRecord
		result2 error
	}
	listBuildsReturnsOnCall map[int]struct {
		result1 []repositories.BuildRecord
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *CFBuildRepository) CreateBuild(arg1 context.Context, arg2 authorization.Info, arg3 repositories.CreateBuildMessage) (repositories.BuildRecord, error) {
	fake.createBuildMutex.Lock()
	ret, specificReturn := fake.createBuildReturnsOnCall[len(fake.createBuildArgsForCall)]
	fake.createBuildArgsForCall = append(fake.createBuildArgsForCall, struct {
		arg1 context.Context
		arg2 authorization.Info
		arg3 repositories.CreateBuildMessage
	}{arg1, arg2, arg3})
	stub := fake.CreateBuildStub
	fakeReturns := fake.createBuildReturns
	fake.recordInvocation("CreateBuild", []interface{}{arg1, arg2, arg3})
	fake.createBuildMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CFBuildRepository) CreateBuildCallCount() int {
	fake.createBuildMutex.RLock()
	defer fake.createBuildMutex.RUnlock()
	return len(fake.createBuildArgsForCall)
}

func (fake *CFBuildRepository) CreateBuildCalls(stub func(context.Context, authorization.Info, repositories.CreateBuildMessage) (repositories.BuildRecord, error)) {
	fake.createBuildMutex.Lock()
	defer fake.createBuildMutex.Unlock()
	fake.CreateBuildStub = stub
}

func (fake *CFBuildRepository) CreateBuildArgsForCall(i int) (context.Context, authorization.Info, repositories.CreateBuildMessage) {
	fake.createBuildMutex.RLock()
	defer fake.createBuildMutex.RUnlock()
	argsForCall := fake.createBuildArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *CFBuildRepository) CreateBuildReturns(result1 repositories.BuildRecord, result2 error) {
	fake.createBuildMutex.Lock()
	defer fake.createBuildMutex.Unlock()
	fake.CreateBuildStub = nil
	fake.createBuildReturns = struct {
		result1 repositories.BuildRecord
		result2 error
	}{result1, result2}
}

func (fake *CFBuildRepository) CreateBuildReturnsOnCall(i int, result1 repositories.BuildRecord, result2 error) {
	fake.createBuildMutex.Lock()
	defer fake.createBuildMutex.Unlock()
	fake.CreateBuildStub = nil
	if fake.createBuildReturnsOnCall == nil {
		fake.createBuildReturnsOnCall = make(map[int]struct {
			result1 repositories.BuildRecord
			result2 error
		})
	}
	fake.createBuildReturnsOnCall[i] = struct {
		result1 repositories.BuildRecord
		result2 error
	}{result1, result2}
}

func (fake *CFBuildRepository) GetBuild(arg1 context.Context, arg2 authorization.Info, arg3 string) (repositories.BuildRecord, error) {
	fake.getBuildMutex.Lock()
	ret, specificReturn := fake.getBuildReturnsOnCall[len(fake.getBuildArgsForCall)]
	fake.getBuildArgsForCall = append(fake.getBuildArgsForCall, struct {
		arg1 context.Context
		arg2 authorization.Info
		arg3 string
	}{arg1, arg2, arg3})
	stub := fake.GetBuildStub
	fakeReturns := fake.getBuildReturns
	fake.recordInvocation("GetBuild", []interface{}{arg1, arg2, arg3})
	fake.getBuildMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CFBuildRepository) GetBuildCallCount() int {
	fake.getBuildMutex.RLock()
	defer fake.getBuildMutex.RUnlock()
	return len(fake.getBuildArgsForCall)
}

func (fake *CFBuildRepository) GetBuildCalls(stub func(context.Context, authorization.Info, string) (repositories.BuildRecord, error)) {
	fake.getBuildMutex.Lock()
	defer fake.getBuildMutex.Unlock()
	fake.GetBuildStub = stub
}

func (fake *CFBuildRepository) GetBuildArgsForCall(i int) (context.Context, authorization.Info, string) {
	fake.getBuildMutex.RLock()
	defer fake.getBuildMutex.RUnlock()
	argsForCall := fake.getBuildArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *CFBuildRepository) GetBuildReturns(result1 repositories.BuildRecord, result2 error) {
	fake.getBuildMutex.Lock()
	defer fake.getBuildMutex.Unlock()
	fake.GetBuildStub = nil
	fake.getBuildReturns = struct {
		result1 repositories.BuildRecord
		result2 error
	}{result1, result2}
}

func (fake *CFBuildRepository) GetBuildReturnsOnCall(i int, result1 repositories.BuildRecord, result2 error) {
	fake.getBuildMutex.Lock()
	defer fake.getBuildMutex.Unlock()
	fake.GetBuildStub = nil
	if fake.getBuildReturnsOnCall == nil {
		fake.getBuildReturnsOnCall = make(map[int]struct {
			result1 repositories.BuildRecord
			result2 error
		})
	}
	fake.getBuildReturnsOnCall[i] = struct {
		result1 repositories.BuildRecord
		result2 error
	}{result1, result2}
}

func (fake *CFBuildRepository) GetLatestBuildByAppGUID(arg1 context.Context, arg2 authorization.Info, arg3 string, arg4 string) (repositories.BuildRecord, error) {
	fake.getLatestBuildByAppGUIDMutex.Lock()
	ret, specificReturn := fake.getLatestBuildByAppGUIDReturnsOnCall[len(fake.getLatestBuildByAppGUIDArgsForCall)]
	fake.getLatestBuildByAppGUIDArgsForCall = append(fake.getLatestBuildByAppGUIDArgsForCall, struct {
		arg1 context.Context
		arg2 authorization.Info
		arg3 string
		arg4 string
	}{arg1, arg2, arg3, arg4})
	stub := fake.GetLatestBuildByAppGUIDStub
	fakeReturns := fake.getLatestBuildByAppGUIDReturns
	fake.recordInvocation("GetLatestBuildByAppGUID", []interface{}{arg1, arg2, arg3, arg4})
	fake.getLatestBuildByAppGUIDMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3, arg4)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CFBuildRepository) GetLatestBuildByAppGUIDCallCount() int {
	fake.getLatestBuildByAppGUIDMutex.RLock()
	defer fake.getLatestBuildByAppGUIDMutex.RUnlock()
	return len(fake.getLatestBuildByAppGUIDArgsForCall)
}

func (fake *CFBuildRepository) GetLatestBuildByAppGUIDCalls(stub func(context.Context, authorization.Info, string, string) (repositories.BuildRecord, error)) {
	fake.getLatestBuildByAppGUIDMutex.Lock()
	defer fake.getLatestBuildByAppGUIDMutex.Unlock()
	fake.GetLatestBuildByAppGUIDStub = stub
}

func (fake *CFBuildRepository) GetLatestBuildByAppGUIDArgsForCall(i int) (context.Context, authorization.Info, string, string) {
	fake.getLatestBuildByAppGUIDMutex.RLock()
	defer fake.getLatestBuildByAppGUIDMutex.RUnlock()
	argsForCall := fake.getLatestBuildByAppGUIDArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3, argsForCall.arg4
}

func (fake *CFBuildRepository) GetLatestBuildByAppGUIDReturns(result1 repositories.BuildRecord, result2 error) {
	fake.getLatestBuildByAppGUIDMutex.Lock()
	defer fake.getLatestBuildByAppGUIDMutex.Unlock()
	fake.GetLatestBuildByAppGUIDStub = nil
	fake.getLatestBuildByAppGUIDReturns = struct {
		result1 repositories.BuildRecord
		result2 error
	}{result1, result2}
}

func (fake *CFBuildRepository) GetLatestBuildByAppGUIDReturnsOnCall(i int, result1 repositories.BuildRecord, result2 error) {
	fake.getLatestBuildByAppGUIDMutex.Lock()
	defer fake.getLatestBuildByAppGUIDMutex.Unlock()
	fake.GetLatestBuildByAppGUIDStub = nil
	if fake.getLatestBuildByAppGUIDReturnsOnCall == nil {
		fake.getLatestBuildByAppGUIDReturnsOnCall = make(map[int]struct {
			result1 repositories.BuildRecord
			result2 error
		})
	}
	fake.getLatestBuildByAppGUIDReturnsOnCall[i] = struct {
		result1 repositories.BuildRecord
		result2 error
	}{result1, result2}
}

func (fake *CFBuildRepository) ListBuilds(arg1 context.Context, arg2 authorization.Info) ([]repositories.BuildRecord, error) {
	fake.listBuildsMutex.Lock()
	ret, specificReturn := fake.listBuildsReturnsOnCall[len(fake.listBuildsArgsForCall)]
	fake.listBuildsArgsForCall = append(fake.listBuildsArgsForCall, struct {
		arg1 context.Context
		arg2 authorization.Info
	}{arg1, arg2})
	stub := fake.ListBuildsStub
	fakeReturns := fake.listBuildsReturns
	fake.recordInvocation("ListBuilds", []interface{}{arg1, arg2})
	fake.listBuildsMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *CFBuildRepository) ListBuildsCallCount() int {
	fake.listBuildsMutex.RLock()
	defer fake.listBuildsMutex.RUnlock()
	return len(fake.listBuildsArgsForCall)
}

func (fake *CFBuildRepository) ListBuildsCalls(stub func(context.Context, authorization.Info) ([]repositories.BuildRecord, error)) {
	fake.listBuildsMutex.Lock()
	defer fake.listBuildsMutex.Unlock()
	fake.ListBuildsStub = stub
}

func (fake *CFBuildRepository) ListBuildsArgsForCall(i int) (context.Context, authorization.Info) {
	fake.listBuildsMutex.RLock()
	defer fake.listBuildsMutex.RUnlock()
	argsForCall := fake.listBuildsArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *CFBuildRepository) ListBuildsReturns(result1 []repositories.BuildRecord, result2 error) {
	fake.listBuildsMutex.Lock()
	defer fake.listBuildsMutex.Unlock()
	fake.ListBuildsStub = nil
	fake.listBuildsReturns = struct {
		result1 []repositories.BuildRecord
		result2 error
	}{result1, result2}
}

func (fake *CFBuildRepository) ListBuildsReturnsOnCall(i int, result1 []repositories.BuildRecord, result2 error) {
	fake.listBuildsMutex.Lock()
	defer fake.listBuildsMutex.Unlock()
	fake.ListBuildsStub = nil
	if fake.listBuildsReturnsOnCall == nil {
		fake.listBuildsReturnsOnCall = make(map[int]struct {
			result1 []repositories.BuildRecord
			result2 error
		})
	}
	fake.listBuildsReturnsOnCall[i] = struct {
		result1 []repositories.BuildRecord
		result2 error
	}{result1, result2}
}

func (fake *CFBuildRepository) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.createBuildMutex.RLock()
	defer fake.createBuildMutex.RUnlock()
	fake.getBuildMutex.RLock()
	defer fake.getBuildMutex.RUnlock()
	fake.getLatestBuildByAppGUIDMutex.RLock()
	defer fake.getLatestBuildByAppGUIDMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *CFBuildRepository) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ handlers.CFBuildRepository = new(CFBuildRepository)
