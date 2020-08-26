package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/els0r/goProbe/pkg/goDB/storage/gpfile"
	"github.com/sirupsen/logrus"
)

func main() {

	var (
		dbPath string
		dryRun bool
	)
	flag.StringVar(&dbPath, "path", "", "Path to legacy goDB")
	flag.BoolVar(&dryRun, "dry-run", true, "Perform a dry-run")
	flag.Parse()

	if dbPath == "" {
		logrus.StandardLogger().Fatal("Path to legacy goDB requried")
	}

	// Get all interfaces
	dirents, err := ioutil.ReadDir(dbPath)
	if err != nil {
		logrus.StandardLogger().Fatal(err)
	}
	for _, dirent := range dirents {
		if !dirent.IsDir() {
			continue
		}

		// Get all date directories (usually days)
		dates, err := ioutil.ReadDir(filepath.Join(dbPath, dirent.Name()))
		if err != nil {
			logrus.StandardLogger().Fatal(err)
		}
		for _, date := range dates {
			if !date.IsDir() {
				continue
			}

			// Get all files in date directory
			files, err := ioutil.ReadDir(filepath.Join(dbPath, dirent.Name(), date.Name()))
			if err != nil {
				logrus.StandardLogger().Fatal(err)
			}
			for _, file := range files {
				fullPath := filepath.Join(dbPath, dirent.Name(), date.Name(), file.Name())
				if filepath.Ext(strings.TrimSpace(fullPath)) != ".gpf" {
					continue
				}

				// Check if the expected header file already exists (and skip, if so)
				if _, err := os.Stat(fullPath + gpfile.HeaderFileSuffix); err == nil {
					logrus.StandardLogger().Infof("File %s lready converted, skipping...", fullPath)
					continue
				}

				if err := convert(fullPath, dryRun); err != nil {
					logrus.StandardLogger().Fatalf("Error converting legacy file %s: %s", fullPath, err)
				}
				logrus.StandardLogger().Infof("Converted legacy file %s", fullPath)
			}
		}
	}
}

func convert(path string, dryRun bool) error {

	// Open the legacy file
	legacyFile, err := NewLegacyGPFile(path)
	if err != nil {
		return err
	}
	defer legacyFile.Close()

	tmpPath := path + ".tmp"
	newFile, err := gpfile.New(tmpPath, gpfile.ModeWrite)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	defer os.Remove(tmpPath + gpfile.HeaderFileSuffix)

	timestamps := legacyFile.GetTimestamps()
	overallSizeWritten := 0
	for _, ts := range timestamps {
		if ts == 0 {
			continue
		}
		block, err := legacyFile.ReadTimedBlock(ts)
		if err != nil {
			if err.Error() == "Incorrect number of bytes read for decompression" {
				logrus.StandardLogger().Warnf("%s for legacy file %s, skippping block for timestamp %v", err, path, time.Unix(ts, 0))
				continue
			}
			return err
		}

		// Cut off the now unneccessary block prefix / suffix
		block = block[8 : len(block)-8]

		if err := newFile.WriteBlock(ts, block); err != nil {
			return err
		}
		overallSizeWritten += len(block)
	}

	if err := newFile.Close(); err != nil {
		return err
	}

	if !dryRun {
		if err := os.Rename(tmpPath+gpfile.HeaderFileSuffix, path+gpfile.HeaderFileSuffix); err != nil {
			return err
		}
		if overallSizeWritten > 0 {
			if err := os.Rename(tmpPath, path); err != nil {
				return err
			}
		} else {
			logrus.StandardLogger().Printf("No flow data detected in file %s, no need to write data file (only header was written), removing legacy data file\n", path)
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	return nil
}
