package filesystem

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

type PosixST80FileSystem struct {
	rootDir   string
	mu        sync.Mutex
	files     map[int]*os.File
	nextFd    int
	lastErrNo int
}

func NewPosixST80FileSystem(rootDir string) *PosixST80FileSystem {
	return &PosixST80FileSystem{
		rootDir: rootDir,
		files:   make(map[int]*os.File),
		nextFd:  10, // Start handles at 10 to avoid conflicting with 0/1/2 or -1
	}
}

func (fs *PosixST80FileSystem) pathForFile(name string) string {
	return filepath.Join(fs.rootDir, name)
}

func (fs *PosixST80FileSystem) setErr(err error) {
	if err == nil {
		fs.lastErrNo = 0
		return
	}
	if sysErr, ok := err.(syscall.Errno); ok {
		fs.lastErrNo = int(sysErr)
	} else {
		fs.lastErrNo = 1
	}
}

func (fs *PosixST80FileSystem) OpenFile(name string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.pathForFile(name)
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		fs.setErr(err)
		return -1
	}
	fd := fs.nextFd
	fs.nextFd++
	fs.files[fd] = file
	return fd
}

func (fs *PosixST80FileSystem) CreateFile(name string) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := fs.pathForFile(name)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fs.setErr(err)
		return -1
	}
	fd := fs.nextFd
	fs.nextFd++
	fs.files[fd] = file
	return fd
}

func (fs *PosixST80FileSystem) CloseFile(handle int) int {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	file, ok := fs.files[handle]
	if !ok {
		return -1
	}
	delete(fs.files, handle)
	err := file.Close()
	if err != nil {
		fs.setErr(err)
		return -1
	}
	return 0
}

func (fs *PosixST80FileSystem) Read(handle int, buffer []byte) int {
	fs.mu.Lock()
	file, ok := fs.files[handle]
	fs.mu.Unlock()

	if !ok {
		return -1
	}
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		fs.setErr(err)
		return -1
	}
	return n
}

func (fs *PosixST80FileSystem) Write(handle int, buffer []byte) int {
	fs.mu.Lock()
	file, ok := fs.files[handle]
	fs.mu.Unlock()

	if !ok {
		return -1
	}
	n, err := file.Write(buffer)
	if err != nil {
		fs.setErr(err)
		return -1
	}
	return n
}

func (fs *PosixST80FileSystem) TruncateTo(handle int, length int) bool {
	fs.mu.Lock()
	file, ok := fs.files[handle]
	fs.mu.Unlock()

	if !ok {
		return false
	}
	err := file.Truncate(int64(length))
	if err != nil {
		fs.setErr(err)
		return false
	}
	return true
}

func (fs *PosixST80FileSystem) FileSize(handle int) int {
	fs.mu.Lock()
	file, ok := fs.files[handle]
	fs.mu.Unlock()

	if !ok {
		return -1
	}
	fi, err := file.Stat()
	if err != nil {
		fs.setErr(err)
		return -1
	}
	return int(fi.Size())
}

func (fs *PosixST80FileSystem) FileFlush(handle int) bool {
	fs.mu.Lock()
	file, ok := fs.files[handle]
	fs.mu.Unlock()

	if !ok {
		return false
	}
	err := file.Sync()
	if err != nil {
		fs.setErr(err)
		return false
	}
	return true
}

func (fs *PosixST80FileSystem) EnumerateFiles(each func(filename string)) {
	entries, err := os.ReadDir(fs.rootDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) > 0 && name[0] != '.' && !entry.IsDir() {
			each(name)
		}
	}
}

func (fs *PosixST80FileSystem) RenameFile(oldName, newName string) bool {
	oldPath := fs.pathForFile(oldName)
	newPath := fs.pathForFile(newName)
	err := os.Rename(oldPath, newPath)
	if err != nil {
		fs.setErr(err)
		return false
	}
	return true
}

func (fs *PosixST80FileSystem) DeleteFile(fileName string) bool {
	path := fs.pathForFile(fileName)
	err := os.Remove(path)
	if err != nil {
		fs.setErr(err)
		return false
	}
	return true
}

func (fs *PosixST80FileSystem) SeekTo(handle int, position int) int {
	fs.mu.Lock()
	file, ok := fs.files[handle]
	fs.mu.Unlock()

	if !ok {
		return -1
	}
	n, err := file.Seek(int64(position), io.SeekStart)
	if err != nil {
		fs.setErr(err)
		return -1
	}
	return int(n)
}

func (fs *PosixST80FileSystem) Tell(handle int) int {
	fs.mu.Lock()
	file, ok := fs.files[handle]
	fs.mu.Unlock()

	if !ok {
		return -1
	}
	n, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		fs.setErr(err)
		return -1
	}
	return int(n)
}

func (fs *PosixST80FileSystem) LastError() int {
	return fs.lastErrNo
}

func (fs *PosixST80FileSystem) ErrorText(code int) string {
	return syscall.Errno(code).Error()
}
