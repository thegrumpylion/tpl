package main

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

var archiveSuffixes = []string{".tar", ".tar.gz", ".tar.bz2", ".tar.xz", ".tgz", ".zip"}

func unpack(src string, dst string) error {

	if strings.HasSuffix(src, ".zip") {
		return unpackZip(src, dst)
	}

	var tf io.Reader

	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	tf = file

	if strings.HasSuffix(src, ".gz") || strings.HasPrefix(src, ".tgz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzReader.Close()
		tf = gzReader
	}

	if strings.HasSuffix(src, ".bz2") {
		tf = bzip2.NewReader(file)
	}

	if strings.HasSuffix(src, ".xz") {
		xzReader, err := xz.NewReader(file)
		if err != nil {
			return err
		}
		tf = xzReader
	}

	return unpackTar(tf, dst)
}

func unpackTar(src io.Reader, dst string) error {

	// Create a new tar reader
	tarReader := tar.NewReader(src)

	// Extract each file from the archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// End of archive
			break
		}
		if err != nil {
			return err
		}

		// Determine the destination file path
		filePath := dst + "/" + header.Name
		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", filePath)
		}

		// Create the destination directory if it doesn't exist
		if header.FileInfo().IsDir() {
			err := os.MkdirAll(filePath, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			continue
		}

		// Create the destination file
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}

		// Copy the file contents from the archive to the destination file
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
		}

		// Set the file permissions to match the archive
		err = os.Chmod(filePath, header.FileInfo().Mode())
		if err != nil {
			return err
		}

		// Close the destination file
		err = file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func unpackZip(src string, dst string) error {
	archive, err := zip.OpenReader(src)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", filePath)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
	return nil
}
