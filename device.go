package podsync

type Device interface {
	IsConnected() bool
	GetFreeSpace() (uint64, error)
	CopyFile(srcPath, podcastName, filename string) (string, error)
	RemoveFile(ipodPath string) error
	FileExists(ipodPath string) bool
	FormatBytes(bytes uint64) string
}
