package ipod

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

type IPod struct {
	mountPath     string
	podcastFolder string
}

func New(mountPath, podcastFolder string) *IPod {
	return &IPod{
		mountPath:     mountPath,
		podcastFolder: podcastFolder,
	}
}

func (i *IPod) IsConnected() bool {
	info, err := os.Stat(i.mountPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (i *IPod) MountPath() string {
	return i.mountPath
}

func (i *IPod) GetFreeSpace() (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(i.mountPath, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

func (i *IPod) FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func (i *IPod) PodcastsPath() string {
	return filepath.Join(i.mountPath, i.podcastFolder)
}

func (i *IPod) EpisodePath(podcastName, filename string) string {
	return filepath.Join(i.PodcastsPath(), sanitizeName(podcastName), filepath.Base(filename))
}

func (i *IPod) CopyFile(srcPath, podcastName, filename string) (string, error) {
	destDir := filepath.Join(i.PodcastsPath(), sanitizeName(podcastName))
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(filename))

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return destPath, nil
}

func (i *IPod) RemoveFile(ipodPath string) error {
	if err := os.Remove(ipodPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file: %w", err)
	}
	return nil
}

func (i *IPod) FileExists(ipodPath string) bool {
	_, err := os.Stat(ipodPath)
	return err == nil
}

func sanitizeName(name string) string {
	result := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch c {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			result = append(result, '_')
		default:
			result = append(result, c)
		}
	}
	return string(result)
}
