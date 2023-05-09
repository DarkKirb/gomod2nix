package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
    "io"
)

type Package struct {
	GoPackagePath string `json:"-"`
	Version       string `json:"version"`
	Hash          string `json:"hash"`
	ReplacedPath  string `json:"replaced,omitempty"`
}

type Sources map[string]string

func populateStruct(path string, data interface{}) {
	pathVal := os.Getenv(path)
	if len(path) == 0 {
		panic(fmt.Sprintf("env var '%s' was unset", path))
	}
	path = pathVal

	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &data)
	if err != nil {
		panic(err)
	}
}

func main() {
	sources := make(Sources)
	pkgs := make(map[string]*Package)

	populateStruct("sourcesPath", &sources)

	populateStruct("jsonPath", &pkgs)

	keys := make([]string, 0, len(pkgs))
	for key := range pkgs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	makeSymlinks(keys, sources)
}

func makeSymlinks(keys []string, sources Sources) {
	// Iterate, in reverse order
	for i := len(keys) - 1; i >= 0; i-- {
		key := keys[i]
		src := sources[key]

		paths := []string{key}

		for _, path := range paths {
			vendorDir := filepath.Join("vendor", filepath.Dir(path))
			if err := os.MkdirAll(vendorDir, 0o755); err != nil {
				panic(err)
			}

			vendorPath := filepath.Join("vendor", path)
			if _, err := os.Stat(vendorPath); err == nil {
				populateVendorPath(vendorPath, src)
				continue
			}

			// If the file doesn't already exist, just create a simple symlink
			err := CopyAnything(src, vendorPath)
			if err != nil {
				panic(err)
			}
		}
	}
}

func CopyAnything(src, dst string) error {
    fmt.Println("CopyAnything:", src, dst)
    fileInfo, err := os.Stat(src)
    if err != nil {
        return err
    }

    switch fileInfo.Mode() & os.ModeType{
    case os.ModeDir:
        if err := CreateIfNotExists(dst, 0755); err != nil {
            return err
        }
        if err := CopyDirectory(src, dst); err != nil {
            return err
        }
    case os.ModeSymlink:
        if err := CopySymLink(src, dst); err != nil {
            return err
        }
    default:
        if err := Copy(src, dst); err != nil {
            return err
        }
    }
    return nil
}

func CopyDirectory(scrDir, dest string) error {
    entries, err := os.ReadDir(scrDir)
    if err != nil {
        return err
    }
    for _, entry := range entries {
        sourcePath := filepath.Join(scrDir, entry.Name())
        destPath := filepath.Join(dest, entry.Name())

        if err := CopyAnything(sourcePath, destPath); err != nil {
            return err
        }
    }
    return nil
}

func Copy(srcFile, dstFile string) error {
    out, err := os.Create(dstFile)
    if err != nil {
        return err
    }

    defer out.Close()

    in, err := os.Open(srcFile)
    defer in.Close()
    if err != nil {
        return err
    }

    _, err = io.Copy(out, in)
    if err != nil {
        return err
    }

    return nil
}

func Exists(filePath string) bool {
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        return false
    }

    return true
}

func CreateIfNotExists(dir string, perm os.FileMode) error {
    if Exists(dir) {
        return nil
    }

    if err := os.MkdirAll(dir, perm); err != nil {
        return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
    }

    return nil
}

func CopySymLink(source, dest string) error {
    link, err := os.Readlink(source)
    if err != nil {
        return err
    }
    return os.Symlink(link, dest)
}


func populateVendorPath(vendorPath string, src string) {
	files, err := os.ReadDir(src)
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		innerSrc := filepath.Join(src, f.Name())
		dst := filepath.Join(vendorPath, f.Name())
		if err := CopyAnything(innerSrc, dst); err != nil {
			// assume it's an existing directory, try to link the directory content instead.
			// TODO should we do this recursively?
			files, err := os.ReadDir(innerSrc)
			if err != nil {
				panic(err)
			}
			for _, f := range files {
				srcFile := filepath.Join(innerSrc, f.Name())
				dstFile := filepath.Join(dst, f.Name())
				if err := CopyAnything(srcFile, dstFile); err != nil {
					fmt.Println("ignoring symlink error", srcFile, dstFile)
				}
			}
		}
	}
}
