package imghash

import (
	"bytes"
	"fmt"
	"image"
	"io/fs"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

var (
	// Returned if the header read from a hash object file is invalid.
	InvalidHeader = errors.New("Invalid Header")

	// The file magic for any file that contains hashes. Abbreviation of Difference Hash Object.
	FileMagic string = "DHO"
)

// Returned if a path extension provided is not an acceptable image, video or directory.
type InvalidExtension struct{ ext string }

func (i InvalidExtension) Error() string { return "Invalid Extension: " + i.ext }

const (
	width, height int = 9, 9
)

var (
	videoExtensions = []string{".mp4", ".mkv", ".avi", ".mpg", "gif"} // GIF should be counted as a video so the 1 frame check doesn't fail
	imageExtensions = []string{".jpg", ".jpeg", ".png", ".webp"}
)

// Generic method to check if an element is in a slice, return false if it is not present.
func inSlice[T comparable](arr []T, elem T) bool {
	for _, v := range arr {
		if v == elem {
			return true
		}
	}
	return false
}

// Compares two files, returning an error if they are not equal explaining the reason.
func Compare(file1 *File, file2 *File) error {
	if file1.Length() != file2.Length() {
		return errors.Errorf("File lengths are different: %d vs %d", file1.Length(), file2.Length())
	}

outer:
	for _, h1 := range file1.hashes {
		for _, h2 := range file2.hashes {
			if h1.VHash == h2.VHash && h1.HHash == h2.HHash {
				continue outer
			}
		}

		return errors.Errorf("Could not find hash in second file: V: %d, H: %d", h1.VHash, h1.HHash)
	}

	return nil
}

func ffmpegRunner(name string, video bool) (*[]Hash, error) {
	filter := fmt.Sprintf("scale=%dx%d:flags=bilinear,format=rgba", width, height)
	if video {
		filter = "fps=12," + filter
	}

	var (
		buf    bytes.Buffer
		errbuf bytes.Buffer
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

	var idx uint32
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

// Walks through a directory and all subdirectories, handling each image and video it finds into a single file
func fromdirectory(dir string) (*[]Hash, error) {
	var hashes []Hash

	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if vid := inSlice(videoExtensions, path); vid || inSlice(imageExtensions, path) {
			h, err := ffmpegRunner(path, vid)
			if err != nil {
				return err // TODO: maybe more descriptive?
			}

			hashes = append(hashes, *h...)
		}
		return nil
	})

	return &hashes, err
}

// NewFromPath returns a new file object with a different configuration based on the path provided:
// If the path provided is a video, the returned file will have one or more hash in it based on the number of frames in the video.
// If the path provided is an image, the returned file will only have one hash in it.
// If the path provided is a directory, the returned file will have hashes of every image/video within the directory, including recursive directories
// Otherwise, the function will return an error of type InvalidExtension
func NewFromPath(path string) (*File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var hashes *[]Hash
	if info.IsDir() {
		hashes, err = fromdirectory(path)
		if err != nil {
			return nil, errors.Wrap(err, "hashing directory")
		}
	} else if n := info.Name(); inSlice(imageExtensions, n) {
		hashes, err = ffmpegRunner(path, false)
		if err != nil {
			return nil, errors.Wrap(err, "hashing image")
		}

		// This shouldn't be possible
		if len(*hashes) != 1 {
			return nil, errors.Errorf("%d hashes created instead of 1", len(*hashes))
		}
	} else if inSlice(videoExtensions, n) {
		hashes, err = ffmpegRunner(path, true)
		if err != nil {
			return nil, errors.Wrap(err, "hashing video")
		}
	} else {
		//return nil, errors.Errorf("Invalid extension on file %q", info.Name())
		return nil, InvalidExtension{ext: info.Name()}
	}

	file := NewFile()
	file.hashes = *hashes
	file.path = path
	file.Deduplicate()

	return file, nil
}
