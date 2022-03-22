package pluginmanager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/merico-dev/stream/cmd/devstream/version"
	"github.com/merico-dev/stream/internal/pkg/configloader"
	"github.com/merico-dev/stream/pkg/util/log"

	"github.com/spf13/viper"
)

func DownloadPlugins(conf *configloader.Config) error {
	// create plugins dir if not exist
	pluginDir := viper.GetString("plugin-dir")
	if pluginDir == "" {
		return fmt.Errorf("plugins directory should not be \"\"")
	}
	log.Infof("Using dir <%s> to store plugins.", pluginDir)

	// download all plugins that don't exist locally
	dc := NewPbDownloadClient()

	for _, tool := range conf.Tools {
		pluginFileName := configloader.GetPluginFileName(&tool)
		pluginMD5FileName := configloader.GetPluginMD5FileName(&tool)
		// plugin does not exist
		if _, err := os.Stat(filepath.Join(pluginDir, pluginFileName)); errors.Is(err, os.ErrNotExist) {
			// download .so file
			if err := dc.download(pluginDir, pluginFileName, tool.Plugin.Version); err != nil {
				return err
			}
			// download .md5 file
			if err := dc.download(pluginDir, pluginMD5FileName, tool.Plugin.Version); err != nil {
				return err
			}
			// check if the downloaded plugin md5 matches with .md5
			if err := checkPluginMismatch(pluginDir, pluginFileName, pluginMD5FileName, tool.Name); err != nil {
				return err
			}
			continue
		}

		// if .so exists
		isMD5Match, err := version.ValidateFileMatchMD5(filepath.Join(pluginDir, pluginFileName), filepath.Join(pluginDir, pluginMD5FileName))
		if err != nil {
			return err
		}
		// if .so exists, and matches with .md5, continue
		if isMD5Match {
			log.Infof("Plugin: %s already exists, no need to download.", pluginFileName)
			continue
		}
		// if .so exists, but mismatches with .md5, re-download
		log.Infof("Plugin: %s doesn't match with dtm core and will be downloaded.", pluginFileName)
		if err := redownloadPlugins(dc, pluginDir, pluginFileName, pluginMD5FileName, tool.Plugin.Version); err != nil {
			return err
		}

		// check if the downloaded plugin md5 matches with .md5
		if err := checkPluginMismatch(pluginDir, pluginFileName, pluginMD5FileName, tool.Name); err != nil {
			return err
		}
	}
	return nil
}

// CheckLocalPlugins check if the local plugins match with dtm core
func CheckLocalPlugins(conf *configloader.Config) error {
	pluginDir := viper.GetString("plugin-dir")
	if pluginDir == "" {
		return fmt.Errorf("plugins directory doesn't exist")
	}

	log.Infof("Using dir <%s> to store plugins.", pluginDir)

	for _, tool := range conf.Tools {
		pluginFileName := configloader.GetPluginFileName(&tool)
		pluginMD5FileName := configloader.GetPluginMD5FileName(&tool)
		if _, err := os.Stat(filepath.Join(pluginDir, pluginFileName)); errors.Is(err, os.ErrNotExist) {
			if err != nil {
				return err
			}
			return fmt.Errorf("plugin %s doesn't exist", tool.Name)
		}
		if _, err := os.Stat(filepath.Join(pluginDir, pluginMD5FileName)); errors.Is(err, os.ErrNotExist) {
			if err != nil {
				return err
			}
			return fmt.Errorf(".md5 file of plugin %s doesn't exist", tool.Name)
		}

		if err := checkPluginMismatch(pluginDir, pluginFileName, pluginMD5FileName, tool.Name); err != nil {
			return err
		}
	}
	return nil
}

// checkPluginMismatch check if the plugins match with .md5
func checkPluginMismatch(pluginDir, soFileName, md5FileName, tooName string) error {
	isMD5Match, err := version.ValidateFileMatchMD5(filepath.Join(pluginDir, soFileName), filepath.Join(pluginDir, md5FileName))
	if err != nil {
		return err
	}
	if !isMD5Match {
		return fmt.Errorf("plugin %s doesn't match with .md5", tooName)
	}
	return nil
}

// redownloadPlugins re-download from remote
func redownloadPlugins(dc *PbDownloadClient, pluginDir, pluginFileName, pluginMD5FileName, version string) error {
	if err := os.Remove(filepath.Join(pluginDir, pluginFileName)); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(pluginDir, pluginMD5FileName)); err != nil {
		return err
	}
	// download .so file
	if err := dc.download(pluginDir, pluginFileName, version); err != nil {
		return err
	}
	// download .md5 file
	if err := dc.download(pluginDir, pluginMD5FileName, version); err != nil {
		return err
	}
	return nil
}

// DownloadDtmMD5 download remote dtm .md5 for compare with local .md5
func DownloadDtmMD5(remoteMD5Dir, dtmFileName string) error {
	dc := NewPbDownloadClient()
	if err := dc.download(remoteMD5Dir, dtmFileName, version.Version); err != nil {
		return err
	}
	return nil
}
