// Copyright 2018 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package stdlib

import (
	"io"
	"io/ioutil"
	"os"

	"github.com/gentee/gentee/core"
)

// InitFile appends stdlib int functions to the virtual machine
func InitFile(vm *core.VirtualMachine) {
	NewStructType(vm, `finfo`, []string{
		`Name:str`, `Size:int`, `Mode:int`,
		`Time:time`, `IsDir:bool`,
	})

	for _, item := range []interface{}{
		AppendFileºStrBuf, // AppendFile( str, buf )
		AppendFileºStrStr, // AppendFile( str, str )
		ChDirºStr,         // ChDir( str )
		CopyFileºStrStr,   // CopyFile( str, str )
		CreateDirºStr,     // CreateDir( str )
		GetCurDir,         // GetCurDir( ) str
		ReadFileºStr,      // ReadFile( str ) str
		ReadFileºStrBuf,   // ReadFile( str, buf ) buf
		RemoveºStr,        // Remove( str )
		RemoveDirºStr,     // RemoveDir( str )
		RenameºStrStr,     // Rename( str, str )
		TempDir,           // TempDir()
		TempDirºStrStr,    // TempDir(str, str)
		WriteFileºStrBuf,  // WriteFile( str, buf )
		WriteFileºStrStr,  // WriteFile( str, str )
	} {
		vm.StdLib().NewEmbed(item)
	}
	for _, item := range []embedInfo{
		{FileInfoºStr, `str`, `finfo`},        // FileInfo( str ) finfo
		{ReadDirºStr, `str`, `arr.finfo`},     // ReadDir( str ) arr.finfo
		{SetFileTimeºStrTime, `str,time`, ``}, // SetFileTime( str, time )
	} {
		vm.StdLib().NewEmbedExt(item.Func, item.InTypes, item.OutType)
	}
}

func appendFile(filename string, data []byte) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func fromFileInfo(fileInfo os.FileInfo, finfo *core.Struct) *core.Struct {
	finfo.Values[0] = fileInfo.Name()
	finfo.Values[1] = fileInfo.Size()
	finfo.Values[2] = fileInfo.Mode()
	fromTime(finfo.Values[3].(*core.Struct), fileInfo.ModTime())
	finfo.Values[4] = fileInfo.IsDir()
	return finfo
}

// AppendFileºStrBuf appends a buffer to a file
func AppendFileºStrBuf(filename string, buf *core.Buffer) error {
	return appendFile(filename, buf.Data)
}

// AppendFileºStrStr appends a string to a file
func AppendFileºStrStr(filename, s string) error {
	return appendFile(filename, []byte(s))
}

// ChDirºStr change the current directory
func ChDirºStr(dirname string) error {
	return os.Chdir(dirname)
}

// CopyFileºStrStr copies a file
func CopyFileºStrStr(src, dest string) (int64, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()
	destFile, err := os.Create(dest)
	if err != nil {
		return 0, err
	}
	defer destFile.Close()
	return io.Copy(destFile, srcFile)
}

// GetCurDir returns the current directory
func GetCurDir() (string, error) {
	return os.Getwd()
}

// CreateDirºStr creates the directory(s)
func CreateDirºStr(dirname string) error {
	return os.MkdirAll(dirname, os.ModePerm)
}

// FileInfoºStr returns the finfo describing the named file.
func FileInfoºStr(rt *core.RunTime, name string) (*core.Struct, error) {
	finfo := core.NewStructObj(rt, `finfo`)
	handle, err := os.Open(name)
	if err != nil {
		return finfo, err
	}
	defer handle.Close()
	fileInfo, err := handle.Stat()
	if err != nil {
		return finfo, err
	}
	return fromFileInfo(fileInfo, finfo), nil
}

// ReadDirºStr reads a directory
func ReadDirºStr(rt *core.RunTime, dirname string) (*core.Array, error) {
	ret := core.NewArray()
	fileList, err := ioutil.ReadDir(dirname)
	if err != nil {
		return ret, err
	}
	for _, fileInfo := range fileList {
		ret.Data = append(ret.Data, fromFileInfo(fileInfo, core.NewStructObj(rt, `finfo`)))
	}
	return ret, nil
}

// ReadFileºStr reads a file
func ReadFileºStr(filename string) (string, error) {
	out, err := ioutil.ReadFile(filename)
	if err != nil {
		return ``, err
	}
	return string(out), nil
}

// ReadFileºStrBuf reads a file to buffer
func ReadFileºStrBuf(filename string, buf *core.Buffer) (*core.Buffer, error) {
	out, err := ioutil.ReadFile(filename)
	if err != nil {
		return buf, err
	}
	buf.Data = out
	return buf, nil
}

// RenameºStrStr renames a file or a directory
func RenameºStrStr(oldname, newname string) error {
	return os.Rename(oldname, newname)
}

// RemoveºStr removes a file or an empty directory
func RemoveºStr(filename string) error {
	return os.Remove(filename)
}

// RemoveDirºStr removes a directory
func RemoveDirºStr(dirname string) error {
	return os.RemoveAll(dirname)
}

// SetFileTimeºStrTime changes the modification time of the named file
func SetFileTimeºStrTime(name string, ftime *core.Struct) error {
	mtime := toTime(ftime)
	return os.Chtimes(name, mtime, mtime)
}

// TempDir returns the temporary directory
func TempDir() string {
	return os.TempDir()
}

// TempDirºStrStr creates a directory in the temporary directory
func TempDirºStrStr(dir, prefix string) (string, error) {
	return ioutil.TempDir(dir, prefix)
}

// WriteFileºStrBuf writes a buffer to a file
func WriteFileºStrBuf(filename string, buf *core.Buffer) error {
	return ioutil.WriteFile(filename, buf.Data, os.ModePerm)
}

// WriteFileºStrStr writes a string to a file
func WriteFileºStrStr(filename, in string) error {
	return ioutil.WriteFile(filename, []byte(in), os.ModePerm)
}
