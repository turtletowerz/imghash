package imghash

import (
	"encoding/binary"
	"io"
	"math"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// A 32 bit index allows for 4,294,967,295 inices, which when divided by 12fps is 357,913,941 seconds (4000 days) of video... it might be enough
const (
	size08 byte = 1 << iota
	size16
	size32
)

// TODO: Consider adding an array of strings at the beginning that has path names, and give each hash a frame index and path index
type File struct {
	version byte
	_       [4]byte // Future additions may require more things to be added, better to put in a few extra bytes here to future-proof
	maxSize byte
	hashes  []Hash
	path    string
}

// Creates a new file with the default file version
func NewFile() *File {
	return &File{version: 1, maxSize: size32}
}

// Creates a new file with the provided version. Currently only "1" exists as a valid option
func NewFileWithVersion(version byte) *File {
	if version != 1 {
		version = 1
	}
	return &File{version: version, maxSize: size32}
}

// Returns the length of the hash array, equivalent to len(*Hashes())
func (f *File) Length() int {
	return len(f.hashes)
}

// Returns a pointer to all the hashes currently in the file
func (f *File) Hashes() *[]Hash {
	return &f.hashes
}

// O(n^2) deduplication of hashes
func (f *File) Deduplicate() {
	var ret []Hash

outer:
	for _, h1 := range f.hashes {
		for _, h2 := range ret {
			if h1.HHash == h2.HHash && h1.VHash == h2.VHash {
				continue outer
			}
		}

		ret = append(ret, h1)
	}

	f.hashes = ret
}

// Writes the file to the given output path, appending the appropriate file extension
func (f *File) Write(path string) error {
	// determine byte size to use
	var maxIndex uint32
	for _, h := range f.hashes {
		if h.Index > maxIndex {
			maxIndex = h.Index
		}
	}

	if maxIndex <= math.MaxUint8 {
		f.maxSize = size08
	} else if maxIndex <= math.MaxUint16 {
		f.maxSize = size16
	}

	// I did this to save on the number of writes and error checking the default binary package does
	// It's also necessary since variable index sizes are allowed
	s := int(f.maxSize)
	buf := make([]byte, 13+(8+8+s)*f.Length()) // 13 = 3 byte file header + version byte + size byte + 4 for useless + 4 for hash length

	copy(buf[:3], FileMagic[:])
	buf[3] = f.version
	buf[4] = byte(f.maxSize)
	binary.LittleEndian.PutUint32(buf[9:], uint32(f.Length()))

	for i, h := range f.hashes {
		t := buf[13+(16+s)*i:]
		switch f.maxSize {
		case size08:
			t[0] = uint8(h.Index)
		case size16:
			binary.LittleEndian.PutUint16(t[0:], uint16(h.Index))
		case size32:
			binary.LittleEndian.PutUint32(t[0:], h.Index)
		default:
			panic("unreachable")
		}

		binary.LittleEndian.PutUint64(t[f.maxSize+0:], h.VHash)
		binary.LittleEndian.PutUint64(t[f.maxSize+8:], h.VHash)
	}

	if err := os.WriteFile(path+"."+strings.ToLower(FileMagic), buf, 0666); err != nil {
		return errors.Wrap(err, "writing output")
	}
	return nil
}

// Read a given file into a hashinfo object returns InvalidHeader
// error if the file type is invalid, or a normal error otherwise
func LoadFromFile(name string) (*File, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	header := make([]byte, 13)
	if _, err := io.ReadFull(file, header); err != nil {
		return nil, errors.Wrap(err, "reading header data")
	}

	if string(header[:3]) != FileMagic {
		return nil, InvalidHeader
	}

	f := new(File)
	f.path = name
	f.version = header[3]
	f.maxSize = header[4]

	count := binary.LittleEndian.Uint32(header[9:])
	f.hashes = make([]Hash, count)

	buf := make([]byte, count*(8+8+uint32(f.maxSize)))
	if _, err := io.ReadFull(file, buf); err != nil {
		if err == io.ErrUnexpectedEOF {
			return nil, errors.New("reached end of file. Maybe the number of frames is incorrect?")
		}
		return nil, errors.Wrap(err, "reading hash data")
	}

	// Doing it manually skips a lot of unnecessary reflection with low complexity addition
	for i := uint32(0); i < count; i++ {
		h := &f.hashes[i]

		switch f.maxSize {
		case size08:
			h.Index = uint32(buf[0])
		case size16:
			h.Index = uint32(binary.LittleEndian.Uint16(buf[0:]))
		case size32:
			h.Index = binary.LittleEndian.Uint32(buf[0:])
		default:
			panic("unreachable")
		}

		h.VHash = binary.LittleEndian.Uint64(buf[f.maxSize+0:])
		h.HHash = binary.LittleEndian.Uint64(buf[f.maxSize+8:])
		buf = buf[f.maxSize+16:]
	}

	return f, nil
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
