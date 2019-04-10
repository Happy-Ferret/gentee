// Copyright 2019 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package core

import (
	"sync"
	//	"reflect"
)

// Thread contains information about a thread
type Thread struct {
	Owner *RunTime // The runtime of the thread
}

// RootThread is a structure for thread management
type RootThread struct {
	ConstMutex sync.RWMutex
	CtxMutex   sync.RWMutex
	RootMutex  sync.RWMutex
	WG         sync.WaitGroup
	Context    map[string]string
	//	Owner *RunTime // The runtime of the thread
}

func newRootThread() (ret *RootThread) {
	ret = &RootThread{
		Context: make(map[string]string),
	}
	return ret
}

func (rt *RunTime) newThread() {
}

// Thread executes a new thread
func (rt *RunTime) Thread(funcObj *FuncObject) {
	thread := &RunTime{
		VM:    rt.VM,
		Stack: make([]interface{}, 0, 1024),
		Calls: make([]ICmd, 0, 64),
		Root:  rt.Root,
		Cycle: rt.Cycle,
		Depth: rt.Depth,
	}
	thread.newThread()
	rt.Root.Threads.WG.Add(1)
	go func() {
		defer rt.Root.Threads.WG.Done()
		if err := thread.runCmd(&funcObj.Block); err != nil {
			return
		}
	}()
}

// GetConst returns the value of the constant
func (rt *RunTime) GetConst(cmd ICmd) (err error) {
	name := cmd.GetObject().GetName()

	rt.Root.Threads.ConstMutex.RLock()
	v, ok := rt.Root.Consts[name]
	if ok {
		rt.Stack = append(rt.Stack, v)
		rt.Root.Threads.ConstMutex.RUnlock()
	} else {
		// TODO: Insert redefinition of constants here
		constObj := cmd.GetObject().(*ConstObject)
		if constObj.Iota != NotIota {
			rt.Root.Consts[ConstIota] = constObj.Iota
		}
		rt.Root.Threads.ConstMutex.RUnlock()
		if err = rt.runCmd(constObj.Exp); err != nil {
			return err
		}
		rt.Root.Threads.ConstMutex.Lock()
		rt.Consts[name] = rt.Stack[len(rt.Stack)-1]
		rt.Root.Threads.ConstMutex.Unlock()
	}
	return nil
}