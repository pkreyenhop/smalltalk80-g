package filesystem

type FileSystem interface {
	OpenFile(name string) int
	CreateFile(name string) int
	CloseFile(handle int) int
	Read(handle int, buffer []byte) int
	Write(handle int, buffer []byte) int
	TruncateTo(handle int, length int) bool
	FileSize(handle int) int
	FileFlush(handle int) bool
	EnumerateFiles(each func(filename string))
	RenameFile(oldName, newName string) bool
	DeleteFile(fileName string) bool
	SeekTo(handle int, position int) int
	Tell(handle int) int
	LastError() int
	ErrorText(code int) string
}
