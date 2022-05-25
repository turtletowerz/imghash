package imghash

import (
	"bytes"
	"fmt"
	"image"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

var (
	InvalidHeader    = errors.New("Invalid Header")
	InvalidExtension = errors.New("Invalid Extension")

	// The file magic for any file that contains hashes
	FileMagic string = "TUR"
)

const (
	width, height int = 9, 9
)

func isVideo(filename string) bool {
	switch filepath.Ext(filename) {
	case ".mp4", ".mkv", ".avi", ".mpg":
		return true
	default:
		return false
	}
}

func isImage(filename string) bool {
	switch filepath.Ext(filename) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}

// Compares two files, returning whether or not that are equal
func Compare(file1 *File, file2 *File) bool {
	if file1.Length() != file2.Length() {
		//buf.WriteString(fmt.Sprintf("File lengths are different: %d vs %d\n", file1.length, file2.length))
		return false
	}

outer:
	for _, h1 := range file1.hashes {
		for _, h2 := range file2.hashes {
			if h1.VHash == h2.VHash && h1.HHash == h2.HHash {
				//buf.WriteString(fmt.Sprintf("Could not find index %d in second file: %d, %d\n", hash1.Index, hash1.VHash, hash1.HHash))
				continue outer
			}
		}
		return false
	}

	// 	if buf.Len() == 0 {
	// 		log.Println("Files have the same hashes")
	// 		return nil
	// 	}
	return true
}

func ffmpegRunner(name string, video bool) (*[]Hash, error) {
	filter := fmt.Sprintf("scale=%dx%d:flags=bilinear,format=rgba", width, height)
	if video {
		filter = "fps=12," + filter
	}

	var (
		buf    bytes.Buffer
		errbuf bytes.Buffer
		idx    uint32
	)

	cmd := exec.Command("ffmpeg", "-hide_banner", "-i", name, "-vf", filter, "-f", "rawvideo", "pipe:1")
	cmd.Stdout = &buf
	cmd.Stderr = &errbuf

	if err := cmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "running command (stderr: %s)", errbuf.String())
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	if buf.Len()%len(img.Pix) != 0 {
		return nil, errors.Errorf("buffer length must be a multiple of image size (%d), but was %d", len(img.Pix), buf.Len())
	}

	hashes := make([]Hash, buf.Len()/len(img.Pix)) // The number of images from FFmpeg's buffer

	for buf.Len() > 0 {
		idx++
		if _, err := buf.Read(img.Pix); err != nil {
			return nil, err
		}

		vh, hh, err := differenceHash(img)
		if err != nil {
			return nil, errors.Wrapf(err, "creating hash for frame %d", idx)
		}

		h := &hashes[idx-1]
		h.VHash = vh
		h.HHash = hh
		h.Index = idx
	}

	return &hashes, nil
}

// New from file returns
func NewFromFile(name string) (*Hash, error) {
	// TODO: how to handle gifs?
	if !isImage(name) {
		return nil, InvalidExtension
	}

	hashes, err := ffmpegRunner(name, false)
	if err != nil {
		return nil, errors.Wrap(err, "creating hash")
	}

	if len(*hashes) != 1 {
		return nil, errors.New("0 or >1 hashes created (wat)")
	}

	// Lol
	return &(*hashes)[0], nil
}

func NewFromVideo(name string) (*File, error) {
	if !isVideo(name) {
		return nil, InvalidExtension
	}

	file := NewFile()
	//file.path = name

	hashes, err := ffmpegRunner(name, true)
	if err != nil {
		return nil, errors.Wrap(err, "hashing video")
	}

	file.hashes = *hashes
	file.Deduplicate()

	return file, nil
}

// Walks through a directory and all subdirectories, handling each image and video it finds into a single file
func NewFromDirectory(dir string) (*File, error) {
	f := new(File)

	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if vid := isVideo(path); vid || isImage(path) {
			hashes, err := ffmpegRunner(path, vid)
			if err != nil {
				return errors.Wrapf(err, "")
			}

			f.hashes = append(f.hashes, *hashes...)
		}
		return nil
	})

	f.Deduplicate()
	return f, err
}
