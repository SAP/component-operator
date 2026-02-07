/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package generator

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/manifests/helm"
	"github.com/sap/component-operator-runtime/pkg/manifests/kustomize"

	"github.com/sap/component-operator/internal/decrypt"
)

type Item struct {
	Generator  manifests.Generator
	ValidUntil time.Time
}

// TODO: make configurable
const validity = 60 * time.Minute

var items map[string]*Item
var mutex sync.Mutex

func init() {
	items = make(map[string]*Item)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			<-ticker.C
			now := time.Now()
			mutex.Lock()
			for id, item := range items {
				if item.ValidUntil.Before(now) {
					delete(items, id)
				}
			}
			mutex.Unlock()
		}
	}()
}

func GetGenerator(url string, path string, digest string, decryptionProvider string, decryptionKeys map[string][]byte) (manifests.Generator, error) {
	mutex.Lock()
	defer mutex.Unlock()

	// note: url is actually not needed in the generator id, digest and path is enough to identify the content
	id := url + "\n" + digest + "\n" + path + "\n" + decryptionProvider + "\n" + calculateDigest(decryptionKeys)

	if item, ok := items[id]; ok {
		item.ValidUntil = time.Now().Add(validity)
		return item.Generator, nil
	} else {
		tmpdir, err := os.MkdirTemp("", "component-operator-")
		if err != nil {
			return nil, err
		}
		defer func() {
			os.RemoveAll(tmpdir)
		}()
		var decryptor manifests.Decryptor
		if len(decryptionKeys) > 0 {
			switch decryptionProvider {
			case "sops", "":
				sopsDecryptor, err := decrypt.NewSopsDecryptor(decryptionKeys)
				if err != nil {
					return nil, err
				}
				defer sopsDecryptor.Cleanup()
				decryptor = sopsDecryptor
			default:
				return nil, fmt.Errorf("invalid decryption provider: %s", decryptionProvider)
			}
		}
		if err := downloadArchive(url, tmpdir); err != nil {
			return nil, err
		}
		fullPath := filepath.Join(tmpdir, path)
		if info, err := os.Stat(fullPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("no such file or directory: %s", path)
			} else {
				return nil, err
			}
		} else if !info.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", path)
		}
		root, err := os.OpenRoot(tmpdir)
		if err != nil {
			return nil, err
		}
		defer root.Close()

		var generator manifests.Generator
		if _, err = root.Stat(filepath.Join(path, "Chart.yaml")); err == nil {
			if err := decryptDirectory(root, path, decryptor); err != nil {
				return nil, err
			}
			generator, err = helm.NewHelmGenerator(root.FS(), path, nil)
			if err != nil {
				return nil, err
			}
		} else if errors.Is(err, fs.ErrNotExist) {
			generator, err = kustomize.NewKustomizeGenerator(root.FS(), path, nil, kustomize.KustomizeGeneratorOptions{Decryptor: decryptor})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
		items[id] = &Item{Generator: generator, ValidUntil: time.Now().Add(validity)}
		return generator, nil
	}
}

func downloadArchive(url string, targetPath string) error {
	// TODO: use a local or even global file cache
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading %s: %s", url, resp.Status)
	}

	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Name == "." {
			continue
		}
		if filepath.IsAbs(header.Name) {
			return fmt.Errorf("archive must not contain entries with absolute paths (%s)", header.Name)
		}
		path := filepath.Clean(header.Name)
		fullPath := filepath.Join(targetPath, path)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(fullPath)
			if err != nil {
				return err
			}
			if err := func() error {
				defer outFile.Close()
				_, err := io.Copy(outFile, tarReader)
				return err
			}(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("encountered unknown tar type while downloading URL: %v in %s", header.Typeflag, path)
		}
	}

	return nil
}

func decryptDirectory(root *os.Root, path string, decryptor manifests.Decryptor) error {
	if decryptor == nil {
		return nil
	}
	return fs.WalkDir(root.FS(), path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			oldData, err := root.ReadFile(p)
			if err != nil {
				return err
			}
			newData, err := decryptor.Decrypt(oldData, p)
			if err != nil {
				return err
			}
			if !bytes.Equal(newData, oldData) {
				if err := root.WriteFile(p, newData, 0); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
