/*
 * BSD 3-Clause License
 * Copyright (c) 2018, Psiphon Inc.
 * All rights reserved.
 */

package multoml

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/imdario/mergo"
	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

// Conf holds the loaded config. It can be accessed with toml.Tree methods.
// See: https://godoc.org/github.com/pelletier/go-toml
type Conf struct {
	toml.Tree
}

// NewFromFiles loads the config from filenames.
// filenames is the names of the files that will contribute to this config. All files will
// be used, and each will be merged on top of the previous ones. The first filename must
// exist, but subsequent names are optional. The intention is that the first file is the
// primary config, and the other files optionally override that.
// searchPaths is the set of paths where the config files will be looked for, in order.
// When the file is found, the search will stop. If this is set to {""}, the filenames
// will be used unmodified. (So, absolute paths could be set in filenames and they will be
// used directly.)
// envOverrides has the format {"DATABASE_HOST": "database.host"} where "DATABASE_HOST" is
// the envrionment variable name and "database.host" is the TOML config key to override.
func NewFromFiles(filenames, searchPaths []string, envOverrides map[string]string) (conf *Conf, filesUsed []string, err error) {
	if len(filenames) == 0 {
		return nil, nil, errors.Errorf("at least one filename must be provided")
	}

	readClosers, filesUsed, err := readClosersFromFiles(filenames, searchPaths)
	if err != nil {
		err = errors.Wrap(err, "readClosersFromFiles failed")
		return nil, nil, err
	}

	defer func() {
		for _, rc := range readClosers {
			if rc != nil {
				rc.Close()
			}
		}
	}()

	if len(readClosers) == 0 || readClosers[0] == nil {
		err = errors.Errorf("first config file must exist: %s", filenames[0])
		return nil, nil, err
	}

	readers := make([]io.Reader, len(readClosers))
	for i := range readClosers {
		readers[i] = readClosers[i]
	}

	conf, err = load(readers, filesUsed, envOverrides)
	if err != nil {
		err = errors.Wrap(err, "conf.load failed")
		return nil, nil, err
	}

	return conf, filesUsed, nil
}

// NewFromReaders loads the config from io.Readers.
// The config in each reader in readers will be merged into the one before. The intention
// is that the first reader is the primary config, and the other readers optionally
// override that.
// envOverrides has the format {"DATABASE_HOST": "database.host"} where "DATABASE_HOST" is
// the envrionment variable name and "database.host" is the TOML config key to override.
func NewFromReaders(readers []io.Reader, envOverrides map[string]string) (conf *Conf, err error) {
	if len(readers) == 0 {
		return nil, errors.Errorf("at least one reader must be provided")
	}

	conf, err = load(readers, nil, envOverrides)
	if err != nil {
		err = errors.Wrap(err, "conf.load failed")
		return nil, err
	}

	return conf, nil
}

func load(readers []io.Reader, readerNames []string, envOverrides map[string]string) (*Conf, error) {
	var confTOML *toml.Tree

	for i, r := range readers {
		if r == nil {
			continue
		}

		newTOML, err := toml.LoadReader(r)
		if err != nil {
			readerName := fmt.Sprintf("reader#%d", i)
			if len(readerNames) > i {
				readerName = readerNames[i]
			}
			errors.Wrapf(err, "failed to load TOML: %s", readerName)
			return nil, err
		}

		confTOML, err = mergeConfig(confTOML, newTOML)
		if err != nil {
			err = errors.Wrap(err, "mergeConfig failed")
			return nil, err
		}
	}

	if confTOML == nil {
		return nil, errors.Errorf("load resulted in nil config")
	}

	// Read and merge environment variable override
	var err error
	confTOML, err = mergeEnvironment(confTOML, envOverrides)
	if err != nil {
		err = errors.Wrap(err, "mergeEnvironment failed")
		return nil, err
	}

	conf := Conf{
		Tree: *confTOML,
	}

	return &conf, nil
}

func readClosersFromFiles(filenames, searchPaths []string) (readClosers []io.ReadCloser, filesUsed []string, err error) {
	readClosers = make([]io.ReadCloser, len(filenames))
	filesUsed = make([]string, len(filenames))

	for i, fname := range filenames {
		for _, path := range searchPaths {
			fpath := filepath.Join(path, fname)
			var f *os.File
			f, err := os.Open(fpath)
			if os.IsNotExist(err) {
				continue
			} else if err != nil {
				for _, rc := range readClosers {
					if rc != nil {
						rc.Close()
					}
				}

				err = errors.Wrapf(err, "file open failed for %s", fpath)
				return nil, nil, err
			}

			readClosers[i] = f
			filesUsed[i] = fpath
			break
		}
	}

	return readClosers, filesUsed, nil
}

// mergeConfig merges override into base and returns the result.
func mergeConfig(base, override *toml.Tree) (*toml.Tree, error) {
	if base == nil {
		return override, nil
	}
	if override == nil {
		return base, nil
	}

	baseMap := base.ToMap()
	overrideMap := override.ToMap()

	err := mergo.Merge(&overrideMap, baseMap)
	if err != nil {
		return nil, errors.Wrap(err, "mergo.Merge failed")
	}

	res, err := toml.TreeFromMap(overrideMap)
	if err != nil {
		return nil, errors.Wrap(err, "toml.TreeFromMap failed")
	}

	return res, nil
}

// mergeEnvironment takes a config and looks for any of a environment variable keys in fromEnv to
// set or override missing or existing configuration values and returns the result.
func mergeEnvironment(config *toml.Tree, envOverrides map[string]string) (*toml.Tree, error) {
	if config == nil {
		return nil, errors.Errorf("config is required")
	}

	envConfig, err := toml.Load("")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create TOML tree")
	}

	for envKey, confKey := range envOverrides {
		val, ok := os.LookupEnv(envKey)
		if ok {
			envConfig.Set(confKey, val)
		}
	}

	return mergeConfig(config, envConfig)
}
