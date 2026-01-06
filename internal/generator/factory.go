/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package generator

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		var decryptor decrypt.Decryptor
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
		if err := downloadArchive(url, path, tmpdir, decryptor); err != nil {
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
		fsys := os.DirFS(fullPath)

		var generator manifests.Generator
		if _, err = fs.Stat(fsys, "Chart.yaml"); err == nil {
			generator, err = helm.NewHelmGenerator(fsys, "", nil)
			if err != nil {
				return nil, err
			}
		} else if errors.Is(err, fs.ErrNotExist) {
			generator, err = kustomize.NewKustomizeGenerator(fsys, "", nil, kustomize.KustomizeGeneratorOptions{})
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

func downloadArchive(url string, prefix string, dir string, decryptor decrypt.Decryptor) error {
	prefix = filepath.Clean(prefix)
	// TODO: check that prefix is a relative path and does not contain ..

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
		if prefix != "." && !strings.HasPrefix(path, prefix) {
			continue
		}
		fullPath := filepath.Join(dir, path)
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
			if decryptor == nil {
				if _, err := io.Copy(outFile, tarReader); err != nil {
					return err
				}
			} else {
				tarBytes, err := io.ReadAll(tarReader)
				if err != nil {
					return err
				}
				outBytes, err := decryptor.Decrypt(tarBytes, path)
				if err != nil {
					return err
				}
				if _, err := outFile.Write(outBytes); err != nil {
					return err
				}
			}
			outFile.Close()
		default:
			return fmt.Errorf("encountered unknown tar type while downloading URL: %v in %s", header.Typeflag, path)
		}
	}
	return nil
}
