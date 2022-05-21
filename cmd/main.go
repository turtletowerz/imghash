package main

import (
	"flag"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/turtletowerz/imghash"
)

var (
	filename = flag.String("f", "", "The name of the file to open for hash testing")
	option   = flag.String("o", "write", "Option to pass to the hasher (defualt write)")
	logfile  = flag.String("l", "-", "The location to send hashing logs to (default stdout)")
	logger   *log.Logger
)

func sub(f, s uint32) uint32 {
	if f > s {
		return f - s
	}
	return s - f
}

func testVPTree(dir string) error {
	var hashes []imghash.Hash
	var random *imghash.Hash
	logger.Println("Starting VP Tree test")
	rand.Seed(time.Now().Unix())

	var count int
	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, info fs.DirEntry, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		data, err := imghash.LoadFromFile(filepath.Join(dir, path))

		// If it tried reading a file that wasn't a hash file, ignore the error
		if err != nil && err != imghash.InvalidHeader {
			return nil
		}

		for _, v := range *data.Hashes() {
			hashes = append(hashes, v)
			count++

			if random == nil && rand.Intn(10000) == 100 {
				logger.Println("Random: " + info.Name())
				random = &v
				logger.Println("\t", random)
				//random.Title = info.Name()
			}
		}
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "collecting hashes")
	}

	logger.Println("Completed hash collection:", count*2, "hashes")

	tree := imghash.NewTree(hashes)
	logger.Println("Completed tree with", count, "nodes")

	q := tree.NearestN(random, 5) //tree.NearestDist(random, 10)
	logger.Println("Completed tree searching")

	// Good idea to sort ones with the same difference by their index based on how close they are to the original
	sort.Slice(q, func(i, j int) bool {
		if q[i].Dist == q[j].Dist {
			return sub(random.Index, q[i].Item.Index) <= sub(random.Index, q[j].Item.Index)
		}
		return q[i].Dist < q[j].Dist
	})

	logger.Println("Closest to", random)
	for _, c := range q {
		logger.Println(c.Dist, c.Item)
	}

	nearest, dist := tree.Nearest(random)
	logger.Println("Completed nearest search")
	logger.Println("Closeset item:", dist, nearest)
	return nil
}

func main() {
	flag.Parse()

	if *logfile == "-" {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		file, err := os.OpenFile(*logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		logger = log.New(file, "", log.LstdFlags)
	}

	if *filename == "" {
		logger.Fatalln()
	}

	/*
		f, _ := os.Create("test.mprof")
		defer f.Close()
		//pprof.StartCPUProfile(f)
		//defer pprof.StopCPUProfile()
		pprof.WriteHeapProfile(f)
	*/

	logger.Println("start")
	switch *option {
	case "write":
		file, err := imghash.NewFromVideo(*filename)
		if err != nil {
			logger.Fatalln("Making hash from video: " + err.Error())
		}

		logger.Println("hashing complete", file.Length())

		if err := file.Write(*filename); err != nil {
			logger.Fatalln("Writing: " + err.Error())
		}
	case "read":
		f, err := imghash.LoadFromFile(*filename + "." + strings.ToLower(imghash.FileMagic))
		if err != nil {
			logger.Fatalln("Reading hash from file: " + err.Error())
		}

		logger.Println(f.Length())

		if err := f.Write("new-" + *filename); err != nil {
			logger.Fatalln("Writing read to file: " + err.Error())
		}

		f2, err := imghash.LoadFromFile("new-" + *filename + "." + strings.ToLower(imghash.FileMagic))
		if err != nil {
			logger.Fatalln("Reading hash from file: " + err.Error())
		}

		logger.Println(f2.Length())
		logger.Println(imghash.Compare(f, f2))
	case "check":
		ep, err := imghash.NewFromVideo(*filename)
		if err != nil {
			logger.Fatalln("Making hash from video: " + err.Error())
		}

		logger.Println("hashing complete", ep.Length())

		if err := ep.Write(*filename); err != nil {
			logger.Fatalln("Writing: " + err.Error())
		}

		// img, err := imghash.NewFromFile(".png")
		// if err != nil {
		// 	logger.Fatalln("Making hash from image: " + err.Error())
		// }

		// logger.Println(img)

	case "vptree":
		if err := testVPTree(*filename); err != nil {
			logger.Fatalln("Testing VP Tree: " + err.Error())
		}
	case "compare":
		// TODO
	default:
		flag.PrintDefaults()
		logger.Fatalln("Invalid option " + *option)
	}
	logger.Println("fin")
}
