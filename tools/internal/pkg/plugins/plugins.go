//
// Copyright (c) 2021, NVIDIA CORPORATION. All rights reserved.
//
// See LICENSE.txt for license information
//

package plugins

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"strings"
)

type Plugins struct {
	ImbalanceDetect func(string, string) error
}

func Load(basedir string, plugins *Plugins) error {
	// Look for potential plugins
	pluginsDir := filepath.Join(basedir, "plugins")
	sharedLibs, err := ioutil.ReadDir(pluginsDir)
	if err != nil {
		fmt.Printf("ERROR: impossible to read %s directory\n", pluginsDir)
		os.Exit(1)
	}
	var pluginFiles []string
	for _, f := range sharedLibs {
		if strings.HasSuffix(f.Name(), ".so") {
			pluginFiles = append(pluginFiles, filepath.Join(pluginsDir, f.Name()))
		}
	}
	// No security for now, we simply try to open all .so files from plugin file
	for _, p := range pluginFiles {
		currentPlugin, err := plugin.Open(p)
		if err == nil {
			// We only care if we are able to open the plugin, otherwise we ignore the file
			symb, err := currentPlugin.Lookup("DetectImbalance")
			if err == nil {
				// Check function signature
				targetFunc, ok := symb.(func(string, string) error)
				if ok {
					log.Printf("[INFO] plugin \"DetectImbalance\" ready")
					plugins.ImbalanceDetect = targetFunc
				}
			}
		}
	}
	return nil
}
