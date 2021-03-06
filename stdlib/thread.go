// Copyright 2018 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package stdlib

import (
	"fmt"

	"github.com/gentee/gentee/core"
)

// InitThread appends stdlib thread functions to the virtual machine
func InitThread(vm *core.VirtualMachine) {
	for _, item := range []embedInfo{
		{AssignºThreadThread, `thread,thread`, `thread`},      // thread = thread
		{AssignAddºArrInt, `arr.thread,thread`, `arr.thread`}, // arr += thread
		{terminateºThread, `thread`, ``},                      // close( thread )
		{resumeºThread, `thread`, ``},                         // resume( thread )
		{suspendºThread, `thread`, ``},                        // suspend( thread )
		{waitºThread, `thread`, ``},                           // wait( thread )
	} {
		vm.StdLib().NewEmbedExt(item.Func, item.InTypes, item.OutType)
	}
}

// AssignºThreadThread assigns one thread to another
func AssignºThreadThread(ptr *interface{}, value int64) int64 {
	*ptr = value
	return (*ptr).(int64)
}

type threadFunc func(root *core.RootThread)

func changeStatus(rt *core.RunTime, threadID int64, todo threadFunc) error {
	root := rt.Root.Threads
	root.ThreadMutex.Lock()
	defer root.ThreadMutex.Unlock()
	if threadID <= 0 || int64(len(root.Threads)) <= threadID {
		return fmt.Errorf(core.ErrorText(core.ErrThreadIndex))
	}
	todo(root)
	return nil
}

// terminateºThread closes the thread
func terminateºThread(rt *core.RunTime, threadID int64) error {
	return changeStatus(rt, threadID, func(root *core.RootThread) {
		if root.Threads[threadID].Status < core.ThFinished {
			root.Threads[threadID].Chan <- core.ThCmdClose
		}
	})
}

// resumeºThread resumes the thread
func resumeºThread(rt *core.RunTime, threadID int64) error {
	return changeStatus(rt, threadID, func(root *core.RootThread) {
		if root.Threads[threadID].Status == core.ThPaused {
			root.Threads[threadID].Chan <- core.ThCmdResume
		}
	})
}

// suspendºThread suspends the thread
func suspendºThread(rt *core.RunTime, threadID int64) error {
	return changeStatus(rt, threadID, func(root *core.RootThread) {
		if root.Threads[threadID].Status < core.ThFinished {
			root.Threads[threadID].Status = core.ThPaused
		}
	})
}

// waitºThread waits for the finish of the thread
func waitºThread(rt *core.RunTime, threadID int64) error {
	return changeStatus(rt, threadID, func(root *core.RootThread) {
		if root.Threads[threadID].Status < core.ThFinished {
			root.Threads[threadID].Notify = append(root.Threads[threadID].Notify, rt.ThreadID)
			rt.Thread.Status = core.ThWait
		}
	})
}
